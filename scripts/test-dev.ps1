param([string]$Fixture = '')
$ErrorActionPreference = 'Stop'
$utf8NoBom = New-Object Text.UTF8Encoding($false)
# Keep Windows PowerShell 5.1 as the CI target; the test host may also be pwsh.
$testPowerShell = (Get-Command powershell.exe -CommandType Application -ErrorAction Stop).Source

$tokens = $null; $parseErrors = $null
$null = [Management.Automation.Language.Parser]::ParseFile((Join-Path $PSScriptRoot 'dev.ps1'), [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count) { throw ($parseErrors | Out-String) }

function Assert-Dev($Condition, [string]$Message) {
    if (-not $Condition) { throw $Message }
}

$testRoot = Join-Path ([IO.Path]::GetTempPath()) ('ct-dev-test-' + [guid]::NewGuid().ToString('N'))
try {
    $testRepo = Join-Path $testRoot 'repo'
    $testScripts = Join-Path $testRepo 'scripts'
    $null = [IO.Directory]::CreateDirectory($testScripts)
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'dev.ps1') -Destination (Join-Path $testScripts 'dev.ps1')
    $consoleSource = Join-Path $testRoot 'console.cs'
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'test-dev-console.cs') -Destination $consoleSource
    if (-not ('CtDevConsole' -as [type])) { Add-Type -Path $consoleSource }
    if (-not $Fixture) {
        $Fixture = Join-Path $testRoot 'fixture.exe'
        Push-Location (Split-Path -Parent $PSScriptRoot)
        try {
            go build -o $Fixture ./cmd/ct
            if ($LASTEXITCODE -ne 0) { throw 'Development test fixture build failed.' }
        } finally { Pop-Location }
    }
    # Child sessions use the real ct binary, but rebuilding is replaced by a
    # fixture copy. The fake editor prevents launching the developer's VS Code.
    [IO.File]::WriteAllText((Join-Path $testRoot 'code.cmd'), "@echo off`r`nexit /b 0`r`n", $utf8NoBom)
    $driver = @'
$ErrorActionPreference = 'Stop'
if ($args.Count -eq 1 -and $args[0] -eq '__input-probe') {
    Add-Type -Path (Join-Path $PSScriptRoot 'console.cs')
    [CtDevConsole]::CaptureFirstInputLine((Join-Path $PSScriptRoot 'first-input.bin'))
    exit 0
}
$global:ctDevTestFixture = '__FIXTURE__'
function go {
    if ($args.Count -ne 4 -or $args[0] -ne 'build' -or $args[1] -ne '-o') { throw 'Unexpected build invocation.' }
    [IO.File]::Copy($global:ctDevTestFixture, [string]$args[2], $true)
    $global:LASTEXITCODE = 0
}
$env:PATH = $PSScriptRoot + ';' + $env:PATH
$names = @('CT_CONFIG_DIR', 'APPDATA', 'LOCALAPPDATA', 'HOME', 'USERPROFILE', 'XDG_CONFIG_HOME')
$saved = @{}
foreach ($name in $names) { $saved[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
$beforeLocation = (Get-Location).Path
& (Join-Path $PSScriptRoot 'repo\scripts\dev.ps1') @args
$ctExit = $LASTEXITCODE
foreach ($name in $names) {
    if ([Environment]::GetEnvironmentVariable($name, 'Process') -cne $saved[$name]) { throw "Environment was changed: $name" }
}
if ((Get-Location).Path -ne $beforeLocation) { throw 'Working directory was not restored.' }
exit $ctExit
'@
    $driver = $driver.Replace('__FIXTURE__', ([IO.Path]::GetFullPath($Fixture)).Replace("'", "''"))
    $driverPath = Join-Path $testRoot 'driver.ps1'
    [IO.File]::WriteAllText($driverPath, $driver, $utf8NoBom)

    function Invoke-DevTest([string[]]$CtArgs, [string]$InputText = '', [switch]$ExistingOverride, [switch]$Console, [int]$ExpectedExit = 0) {
        if ($Console) {
            # Keyboard events reach an actual console input buffer, not a pipe.
            # This catches the console-only EOF leak missed by redirected tests.
            $consoleDriver = Join-Path $testRoot 'console-driver.ps1'
            $encoded = [Convert]::ToBase64String($utf8NoBom.GetBytes($InputText))
            $prefix = "Add-Type -Path (Join-Path `$PSScriptRoot 'console.cs')`r`n" +
                "[CtDevConsole]::QueueInput((New-Object Text.UTF8Encoding(`$false)).GetString([Convert]::FromBase64String('$encoded')))`r`n"
            if ($ExistingOverride) {
                $prefix += "`$env:CT_CONFIG_DIR = Join-Path `$PSScriptRoot 'original-config'`r`n"
            } else {
                $prefix += "[Environment]::SetEnvironmentVariable('CT_CONFIG_DIR', `$null, 'Process')`r`n"
            }
            [IO.File]::WriteAllText($consoleDriver, $prefix + $driver, $utf8NoBom)
            $allArgs = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $consoleDriver) + $CtArgs
            $arguments = ($allArgs | ForEach-Object { '"' + $_ + '"' }) -join ' '
            $code = [CtDevConsole]::RunHidden($testPowerShell, $arguments, $testRoot)
            Assert-Dev ($code -eq $ExpectedExit) "Console development command failed: $CtArgs (exit $code)"
            return
        }
        $start = New-Object Diagnostics.ProcessStartInfo
        $start.FileName = $testPowerShell
        $allArgs = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $driverPath) + $CtArgs
        $start.Arguments = ($allArgs | ForEach-Object { '"' + $_ + '"' }) -join ' '
        $start.UseShellExecute = $false
        $start.CreateNoWindow = $true
        $start.RedirectStandardInput = $true
        $start.RedirectStandardOutput = $true
        $start.RedirectStandardError = $true
        if ($ExistingOverride) {
            $start.EnvironmentVariables['CT_CONFIG_DIR'] = Join-Path $testRoot 'original-config'
        } else {
            $start.EnvironmentVariables.Remove('CT_CONFIG_DIR')
        }
        $process = New-Object Diagnostics.Process
        $process.StartInfo = $start
        try {
            [CtDevConsole]::StartWithUtf8Input($process)
            $stdout = $process.StandardOutput.ReadToEndAsync()
            $stderr = $process.StandardError.ReadToEndAsync()
            $process.StandardInput.Write($InputText)
            $process.StandardInput.Close()
            if (-not $process.WaitForExit(30000)) { $process.Kill(); throw "Development command timed out: $CtArgs" }
            $output = $stdout.GetAwaiter().GetResult() + $stderr.GetAwaiter().GetResult()
            Assert-Dev ($process.ExitCode -eq $ExpectedExit) "Development command failed: $CtArgs`n$output"
            return $output
        } finally { $process.Dispose() }
    }

    # Reproduce the CI host's BOM-emitting UTF-8 default deliberately. The
    # receiver must see exactly kotlin, with no BOM silently stripped by a reader.
    $previousInputEncoding = [Console]::InputEncoding
    try {
        [Console]::InputEncoding = New-Object Text.UTF8Encoding($true)
        $null = Invoke-DevTest -CtArgs @('__input-probe') -InputText "kotlin`n"
        $firstInput = $utf8NoBom.GetString([IO.File]::ReadAllBytes((Join-Path $testRoot 'first-input.bin')))
        Assert-Dev (-not $firstInput.StartsWith([string][char]0xFEFF, [StringComparison]::Ordinal)) 'First input contains a UTF-8 BOM.'
        Assert-Dev ([string]::Equals($firstInput, 'kotlin', [StringComparison]::Ordinal)) "First input must be exactly kotlin, got <$firstInput>."
        Assert-Dev ([Console]::InputEncoding.GetPreamble().Length -eq 3) 'Test host input encoding was not restored.'
    } finally { [Console]::InputEncoding = $previousInputEncoding }

    $devConfig = Join-Path $testRepo '.tmp-ct-config'
    $configFile = Join-Path $devConfig 'config.json'
    $devRoot = Join-Path $testRepo '.tmp-coding-test'
    $devExe = Join-Path $testRepo 'ct.exe'

    # Each command starts with no config, bootstraps, and consumes keyboard
    # input in that SAME invocation. Repeat config/remove with a retained registry.
    foreach ($scenario in @('language', 'platform', 'config')) {
        $null = Invoke-DevTest -CtArgs @('--clean')
        Assert-Dev (-not (Test-Path -LiteralPath $configFile)) 'Console test did not start clean.'
        switch ($scenario) {
            'language' {
                $null = Invoke-DevTest -Console -CtArgs @('language', 'add') -InputText "`nkotlin`nkt`n`nkt`nMain.kt`n`nSolution.kt`n`n"
                $registry = Get-Content -LiteralPath (Join-Path $devConfig 'registry.json') -Raw | ConvertFrom-Json
                Assert-Dev ($registry.languages[0].name -eq 'kotlin') 'Console language Name was lost after bootstrap.'
            }
            'platform' {
                $null = Invoke-DevTest -Console -CtArgs @('platform', 'add') -InputText "`nbaekjoon`nboj`n`n`n`n`n`n`n`n`n" -ExistingOverride
                $registry = Get-Content -LiteralPath (Join-Path $devConfig 'registry.json') -Raw | ConvertFrom-Json
                Assert-Dev ($registry.platforms[0].name -eq 'baekjoon') 'Console platform Name was lost after bootstrap.'
            }
            'config' {
                $null = Invoke-DevTest -Console -CtArgs @('config') -InputText "`npg`njava`n`n"
                $cfg = Get-Content -LiteralPath $configFile -Raw | ConvertFrom-Json
                Assert-Dev ($cfg.root -ceq $devRoot -and $cfg.language -eq 'java' -and $cfg.platform -eq 'programmers') 'Console config answers were lost after bootstrap.'
            }
        }
        Assert-Dev ([IO.File]::Exists($configFile)) 'Console invocation did not bootstrap config.'
    }
    $null = Invoke-DevTest -CtArgs @('--clean')

    # First invocation is interactive: its Name must come from the caller's
    # stdin, while only the separate config bootstrap gets synthetic answers.
    $output = Invoke-DevTest -CtArgs @('language', 'add') -InputText "kotlin`nkt`nkt`nMain.kt`n`nSolution.kt`n`n"
    Assert-Dev ($output -match 'Added user language kotlin') 'First language add did not receive caller stdin.'
    $cfg = Get-Content -LiteralPath $configFile -Raw | ConvertFrom-Json
    Assert-Dev ($cfg.root -ceq $devRoot -and $cfg.platform -eq 'codeforces' -and $cfg.language -eq 'cpp') 'Wrong development bootstrap defaults.'
    $originalConfig = [IO.File]::ReadAllText($configFile)

    $output = Invoke-DevTest -CtArgs @('language', 'list')
    Assert-Dev ($output -match 'kotlin' -and $output -match 'Languages') 'Noninteractive registry list failed.'
    # The fake VS Code CLI returns no extensions, so doctor must report issues
    # regardless of the machine's installed compilers; it must not prompt.
    $output = Invoke-DevTest -CtArgs @('doctor') -ExpectedExit 1
    Assert-Dev ($output -match 'Coding Test Environment' -and $output -match 'Environment has') 'Doctor output/exit status was not preserved.'

    $null = Invoke-DevTest -CtArgs @('71A') -ExistingOverride
    $null = Invoke-DevTest -CtArgs @('71A', '-l', 'java')
    Assert-Dev ([IO.File]::Exists((Join-Path $devRoot 'codeforces\71A\main.cpp'))) 'Problem command failed.'
    Assert-Dev ([IO.File]::Exists((Join-Path $devRoot 'codeforces\71A\Main.java'))) 'Problem language arguments were lost.'
    Assert-Dev ([IO.File]::ReadAllText($configFile) -ceq $originalConfig) 'Existing isolated config was rewritten.'

    $output = Invoke-DevTest -CtArgs @('platform', 'add') -InputText "baekjoon`nboj`n`n`n`n`n`n`n`n`nMain.kt`n`n"
    Assert-Dev ($output -match 'Added user platform baekjoon') 'Platform add did not receive caller stdin.'
    $null = Invoke-DevTest -CtArgs @('1000', '-p', 'boj', '-l', 'kt')
    $solution = Join-Path $devRoot 'baekjoon\1000\Main.kt'
    Assert-Dev ([IO.File]::Exists($solution)) 'Custom problem arguments did not work.'

    $null = Invoke-DevTest -CtArgs @('config') -InputText "`nboj`nkt`n`n" -ExistingOverride
    $cfg = Get-Content -LiteralPath $configFile -Raw | ConvertFrom-Json
    Assert-Dev ($cfg.root -ceq $devRoot -and $cfg.platform -eq 'baekjoon' -and $cfg.language -eq 'kotlin') 'Config command did not receive caller stdin.'
    foreach ($kind in @('language', 'platform')) {
        $name = 'kt'; if ($kind -eq 'platform') { $name = 'boj' }
        $output = Invoke-DevTest -CtArgs @($kind, 'remove', $name) -InputText "n`n"
        Assert-Dev ($output -match 'Removal cancelled') 'Removal confirmation did not receive caller stdin.'
    }

    # Keep the registry, remove only config, then confirm removal using console
    # keyboard input immediately after automatic bootstrap. A default-No EOF
    # cancellation would incorrectly leave the entry and fail these assertions.
    foreach ($kind in @('language', 'platform')) {
        [IO.File]::Delete($configFile)
        $name = 'kt'; if ($kind -eq 'platform') { $name = 'boj' }
        $null = Invoke-DevTest -Console -CtArgs @($kind, 'remove', $name) -InputText "y`n" -ExistingOverride
        $registry = Get-Content -LiteralPath (Join-Path $devConfig 'registry.json') -Raw | ConvertFrom-Json
        if ($kind -eq 'language') {
            Assert-Dev (@($registry.languages | Where-Object { $null -ne $_ }).Count -eq 0) 'Console language removal received EOF instead of confirmation.'
        } else {
            Assert-Dev (@($registry.platforms | Where-Object { $null -ne $_ }).Count -eq 0) 'Console platform removal received EOF instead of confirmation.'
        }
        Assert-Dev ([IO.File]::Exists($solution)) 'Console removal deleted an existing solution.'
    }

    $null = Invoke-DevTest -Console -CtArgs @('config') -InputText "`npg`npy`n`n"
    $cfg = Get-Content -LiteralPath $configFile -Raw | ConvertFrom-Json
    Assert-Dev ($cfg.platform -eq 'programmers' -and $cfg.language -eq 'python') 'Existing-config console invocation lost stdin.'

    $null = Invoke-DevTest -CtArgs @('--reset', '71A') -ExistingOverride
    Assert-Dev (-not [IO.File]::Exists($solution)) 'Reset did not clear the isolated solution directory.'
    Assert-Dev ([IO.File]::Exists((Join-Path $devRoot 'codeforces\71A\main.cpp'))) 'Reset did not bootstrap and execute the requested command.'
    Assert-Dev (-not [IO.File]::Exists((Join-Path $devConfig 'registry.json'))) 'Reset retained the previous isolated registry.'

    $sentinel = Join-Path $testRepo 'preserved.txt'
    [IO.File]::WriteAllText($sentinel, 'keep', $utf8NoBom)
    $null = Invoke-DevTest -CtArgs @('--clean') -ExistingOverride
    Assert-Dev (-not (Test-Path -LiteralPath $devConfig) -and -not (Test-Path -LiteralPath $devRoot) -and -not (Test-Path -LiteralPath $devExe)) 'Clean left development artifacts.'
    Assert-Dev ([IO.File]::ReadAllText($sentinel) -ceq 'keep') 'Clean touched an unrelated file.'
    Write-Host 'Development helper tests passed (temporary repository, real ct, isolated config and mocked build/editor).'
} finally {
    $resolved = [IO.Path]::GetFullPath($testRoot)
    $tempBase = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\')
    if ([IO.Path]::GetDirectoryName($resolved) -ne $tempBase -or -not [IO.Path]::GetFileName($resolved).StartsWith('ct-dev-test-')) {
        throw 'Unsafe development test cleanup path.'
    }
    if (Test-Path -LiteralPath $resolved) { Remove-Item -LiteralPath $resolved -Recurse -Force }
}
