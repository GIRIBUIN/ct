param([string]$ScriptsDir = $PSScriptRoot)
$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent $ScriptsDir
foreach ($file in @('install.ps1', 'uninstall.ps1', 'test-distribution.ps1')) {
    $tokens = $null; $parseErrors = $null
    $null = [Management.Automation.Language.Parser]::ParseFile((Join-Path $ScriptsDir $file), [ref]$tokens, [ref]$parseErrors)
    if ($parseErrors.Count) { throw ($parseErrors | Out-String) }
}
. ([scriptblock]::Create([IO.File]::ReadAllText((Join-Path $ScriptsDir 'install.ps1'))))
. ([scriptblock]::Create([IO.File]::ReadAllText((Join-Path $ScriptsDir 'uninstall.ps1'))))
function Assert-Equal($Actual, $Expected) {
    if ($Actual -cne $Expected) { throw "Expected <$Expected>, got <$Actual>" }
}
function Assert-Throws([scriptblock]$Action) {
    $failed = $false
    try { & $Action } catch { $failed = $true }
    if (-not $failed) { throw 'Expected failure.' }
}

$testRoot = Join-Path ([IO.Path]::GetTempPath()) ('ct-dist-test-' + [Guid]::NewGuid().ToString('N'))
$saved = @{}
foreach ($name in @('LOCALAPPDATA', 'APPDATA', 'CT_CONFIG_DIR', 'PATH', 'PROCESSOR_ARCHITECTURE', 'PROCESSOR_ARCHITEW6432')) {
    $saved[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
try {
    $null = [IO.Directory]::CreateDirectory($testRoot)
    $env:LOCALAPPDATA = Join-Path $testRoot 'local'
    $env:APPDATA = Join-Path $testRoot 'roaming'
    $env:CT_CONFIG_DIR = Join-Path $testRoot 'solutions'
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'
    $env:PROCESSOR_ARCHITEW6432 = ''
    $fixture = Join-Path $testRoot 'fixture.exe'
    Push-Location $repo
    try {
        go build -o $fixture ./cmd/ct
        if ($LASTEXITCODE -ne 0) { throw 'Fixture build failed.' }
    } finally { Pop-Location }
    $script:fakeUserPath = 'C:\existing;C:\Other Tool;'
    $script:badHash = $false
    function Get-CtUserPath { return $script:fakeUserPath }
    function Set-CtUserPath([string]$Value) { $script:fakeUserPath = $Value }
    function Get-CtRemovalUserPath { return $script:fakeUserPath }
    function Set-CtRemovalUserPath([string]$Value) { $script:fakeUserPath = $Value }
    $script:interactive = $false
    $script:onboardingCalls = @()
    $script:configExit = 0
    $script:doctorExit = 0
    function Test-CtInteractiveConsole { return $script:interactive }
    function Invoke-CtOnboardingCommand([string]$Target, [string]$Command) {
        Assert-Equal $Target (Join-Path $env:LOCALAPPDATA 'Programs\ct\ct.exe')
        Assert-Equal (Test-Path -LiteralPath $Target) $true
        Assert-Equal ([string]$env:CT_CONFIG_DIR) ''
        $script:onboardingCalls += $Command
        switch ($Command) {
            'config' { return $script:configExit }
            'doctor' { return $script:doctorExit }
            default { throw "Unexpected automatic command: $Command" }
        }
    }
    function Receive-CtFile([string]$Url, [string]$Destination) {
        if (-not $Url.StartsWith('https://github.com/GIRIBUIN/ct/releases/latest/download/')) { throw 'Unexpected download URL.' }
        if ($Url.EndsWith('/checksums.txt')) {
            $hash = (Get-FileHash -LiteralPath $fixture -Algorithm SHA256).Hash
            if ($script:badHash) { $hash = '0' * 64 }
            [IO.File]::WriteAllText($Destination, "$hash  ct-windows-amd64.exe`n")
        } else { [IO.File]::Copy($fixture, $Destination, $true) }
    }
    Assert-Equal (Get-CtAsset 'amd64') 'ct-windows-amd64.exe'
    Assert-Throws { Get-CtAsset 'ARM64' }
    Assert-Equal (Add-CtPath 'C:\one;' 'C:\tool bin') 'C:\one;;C:\tool bin'
    Assert-Equal (Add-CtPath 'C:\ONE;"C:\tool bin\"' 'c:/TOOL BIN') 'C:\ONE;"C:\tool bin\"'
    Assert-Throws { Add-CtPath 'C:\one' 'C:\bad"quote' }
    Assert-Equal (Remove-CtPath 'C:\one;C:\ct;C:\ct-extra;;' 'c:/CT/') 'C:\one;C:\ct-extra;;'
    Assert-Equal (Get-CtCanonicalConfig $env:APPDATA) (Join-Path $env:APPDATA 'ct')
    Assert-Throws { Get-CtCanonicalConfig 'relative' }

    $manifest = Join-Path $testRoot 'checksums.txt'
    $hash = (Get-FileHash -LiteralPath $fixture -Algorithm SHA256).Hash
    [IO.File]::WriteAllText($manifest, "$hash  ct-windows-amd64.exe.extra`n")
    Assert-Throws { Assert-CtChecksum $fixture $manifest 'ct-windows-amd64.exe' }
    [IO.File]::WriteAllText($manifest, "$hash  ct-windows-amd64.exe`n$hash  ct-windows-amd64.exe`n")
    Assert-Throws { Assert-CtChecksum $fixture $manifest 'ct-windows-amd64.exe' }

    Install-Ct
    Assert-Equal $script:onboardingCalls.Count 0
    $installed = Join-Path $env:LOCALAPPDATA 'Programs\ct\ct.exe'
    $firstPath = $script:fakeUserPath
    Install-Ct
    Assert-Equal $script:fakeUserPath $firstPath
    Assert-Equal (Get-FileHash -LiteralPath $installed).Hash $hash
    $script:badHash = $true
    Assert-Throws { Install-Ct }
    Assert-Equal (Get-FileHash -LiteralPath $installed).Hash $hash
    Assert-Equal $script:fakeUserPath $firstPath

    $canonical = Get-CtCanonicalConfig $env:APPDATA
    $null = [IO.Directory]::CreateDirectory($canonical)
    $null = [IO.Directory]::CreateDirectory($env:CT_CONFIG_DIR)
    # An override config must not suppress fresh canonical onboarding.
    [IO.File]::WriteAllText((Join-Path $env:CT_CONFIG_DIR 'config.json'), 'override preserved')
    $script:badHash = $false
    $script:interactive = $true
    foreach ($scenario in @('fresh', 'config-failed', 'doctor-failed', 'existing')) {
        $script:onboardingCalls = @()
        $script:configExit = 0
        $script:doctorExit = 0
        if ($scenario -eq 'config-failed') { $script:configExit = 1 }
        if ($scenario -eq 'doctor-failed') { $script:doctorExit = 1 }
        if ($scenario -eq 'existing') { [IO.File]::WriteAllText((Join-Path $canonical 'config.json'), 'existing preserved') }
        $output = @(Install-Ct 6>&1)
        Assert-Equal (@($output | Where-Object { "$_" -eq 'ct dev' }).Count) 1
        $expected = 'config,doctor'
        if ($scenario -eq 'config-failed') { $expected = 'config' }
        if ($scenario -eq 'existing') { $expected = '' }
        Assert-Equal ($script:onboardingCalls -join ',') $expected
        Assert-Equal (Get-FileHash -LiteralPath $installed).Hash $hash
        Assert-Equal $env:CT_CONFIG_DIR (Join-Path $testRoot 'solutions')
        if ($scenario -eq 'doctor-failed' -and ($output -join "`n") -notmatch 'ct setup --dry-run') { throw 'Missing setup guidance.' }
    }
    Assert-Equal ([IO.File]::ReadAllText((Join-Path $canonical 'config.json'))) 'existing preserved'
    Assert-Equal ([IO.File]::ReadAllText((Join-Path $env:CT_CONFIG_DIR 'config.json'))) 'override preserved'
    [IO.File]::WriteAllText((Join-Path $canonical 'config.json'), '{}')
    [IO.File]::WriteAllText((Join-Path $canonical 'main.cpp'), 'keep canonical sibling')
    [IO.File]::WriteAllText((Join-Path $env:CT_CONFIG_DIR 'main.cpp'), 'keep solutions')
    Uninstall-Ct
    Assert-Equal (Test-Path -LiteralPath $installed) $false
    Assert-Equal $script:fakeUserPath 'C:\existing;C:\Other Tool;'
    Assert-Equal (Test-Path -LiteralPath (Join-Path $canonical 'config.json')) $true
    Uninstall-Ct -RemoveConfig
    Assert-Equal (Test-Path -LiteralPath (Join-Path $canonical 'config.json')) $false
    Assert-Equal ([IO.File]::ReadAllText((Join-Path $canonical 'main.cpp'))) 'keep canonical sibling'
    Assert-Equal ([IO.File]::ReadAllText((Join-Path $env:CT_CONFIG_DIR 'main.cpp'))) 'keep solutions'
    # Exercise the README invocation forms with only the original entry guards and stubs.
    $installAst = [Management.Automation.Language.Parser]::ParseFile((Join-Path $ScriptsDir 'install.ps1'), [ref]$tokens, [ref]$parseErrors)
    $uninstallAst = [Management.Automation.Language.Parser]::ParseFile((Join-Path $ScriptsDir 'uninstall.ps1'), [ref]$tokens, [ref]$parseErrors)
    function Install-Ct { return 'install entry' }
    function Uninstall-Ct([switch]$RemoveConfig) { return "purge=$RemoveConfig" }
    Assert-Equal ($installAst.EndBlock.Statements[-1].Extent.Text | Invoke-Expression) 'install entry'
    $purgeEntry = $uninstallAst.ParamBlock.Extent.Text + "`n" + $uninstallAst.EndBlock.Statements[-1].Extent.Text
    Assert-Equal (& ([scriptblock]::Create($purgeEntry)) -Purge) 'purge=True'
    Write-Host 'Windows distribution tests passed (temporary files, mocked downloads and User PATH).'
} finally {
    foreach ($name in $saved.Keys) { [Environment]::SetEnvironmentVariable($name, $saved[$name], 'Process') }
    $resolved = [IO.Path]::GetFullPath($testRoot)
    $tempBase = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\')
    if ([IO.Path]::GetDirectoryName($resolved) -ne $tempBase -or -not ([IO.Path]::GetFileName($resolved).StartsWith('ct-dist-test-'))) {
        throw 'Unsafe test cleanup path.'
    }
    if (Test-Path -LiteralPath $resolved) { Remove-Item -LiteralPath $resolved -Recurse -Force }
}
