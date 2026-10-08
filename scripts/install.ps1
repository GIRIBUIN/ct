# Self-contained so this script also works through Invoke-RestMethod | Invoke-Expression.
function Get-CtAsset([string]$Architecture) {
    if ($Architecture -ine 'AMD64') { throw "Unsupported Windows architecture: $Architecture (requires amd64)." }
    return 'ct-windows-amd64.exe'
}

function Get-CtPathKey([string]$Value) {
    $value = [Environment]::ExpandEnvironmentVariables($Value.Trim().Trim('"')).Replace('/', '\')
    if (-not $value) { return '' }
    try { $value = [IO.Path]::GetFullPath($value) } catch { }
    return $value.TrimEnd('\').ToLowerInvariant()
}

function Add-CtPath([string]$Current, [string]$Entry) {
    if (-not $Entry -or $Entry.IndexOfAny([char[]]";`"`r`n") -ge 0) { throw 'Unsafe ct PATH entry.' }
    foreach ($part in $Current.Split(';')) {
        if ((Get-CtPathKey $part) -eq (Get-CtPathKey $Entry)) { return $Current }
    }
    if (-not $Current) { return $Entry }
    return $Current + ';' + $Entry
}

function Assert-CtPlainPath([string]$Path) {
    $cursor = [IO.Path]::GetFullPath($Path)
    while ($cursor) {
        if (Test-Path -LiteralPath $cursor) {
            $item = Get-Item -LiteralPath $cursor -Force
            if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) { throw "Refusing linked path: $cursor" }
        }
        $cursor = [IO.Path]::GetDirectoryName($cursor)
    }
}

function Assert-CtChecksum([string]$Binary, [string]$Checksums, [string]$Asset) {
    $hashes = @()
    foreach ($line in [IO.File]::ReadAllLines($Checksums)) {
        if ($line -cmatch '^([0-9a-fA-F]{64}) [ *](.+)$' -and $Matches[2] -ceq $Asset) { $hashes += $Matches[1] }
    }
    if ($hashes.Count -ne 1) { throw "Expected exactly one checksum for $Asset." }
    $actual = (Get-FileHash -LiteralPath $Binary -Algorithm SHA256).Hash
    if ($actual -ine $hashes[0]) { throw "SHA-256 mismatch for $Asset; existing installation was not changed." }
}

function Get-CtUserPath { return [Environment]::GetEnvironmentVariable('Path', 'User') }
function Set-CtUserPath([string]$Value) { [Environment]::SetEnvironmentVariable('Path', $Value, 'User') }
function Receive-CtFile([string]$Url, [string]$Destination) {
    Invoke-WebRequest -UseBasicParsing -Uri $Url -OutFile $Destination
}

function Test-CtInteractiveConsole {
    try { return [Environment]::UserInteractive -and -not [Console]::IsInputRedirected -and -not [Console]::IsOutputRedirected }
    catch { return $false }
}

function Show-CtNextSteps {
    Write-Host "Next:`n  ct config`n  ct doctor`n  ct setup --dry-run"
}

function Invoke-CtOnboardingCommand([string]$Target, [string]$Command) {
    & $Target $Command | Out-Host
    $result = $LASTEXITCODE
    # A failed optional diagnosis must not leave installation looking unsuccessful.
    $global:LASTEXITCODE = 0
    return $result
}

function Start-CtOnboarding([string]$Target) {
    if (-not $env:APPDATA -or -not [IO.Path]::IsPathRooted($env:APPDATA)) {
        Write-Host 'ct is installed; canonical configuration location is unavailable.'
        Show-CtNextSteps
        return
    }
    if (Test-Path -LiteralPath (Join-Path $env:APPDATA 'ct\config.json')) { return }
    if (-not (Test-CtInteractiveConsole)) {
        Show-CtNextSteps
        return
    }
    $previousConfigDir = $env:CT_CONFIG_DIR
    $phase = 'config'
    try {
        # Installer onboarding always uses canonical configuration, not a development override.
        $env:CT_CONFIG_DIR = $null
        if ((Invoke-CtOnboardingCommand $Target 'config') -ne 0) {
            Write-Host 'ct remains installed. Configuration was cancelled or failed.'
            Show-CtNextSteps
            return
        }
        $phase = 'doctor'
        if ((Invoke-CtOnboardingCommand $Target 'doctor') -ne 0) {
            Write-Host "ct is installed, but some coding-test environment checks failed.`nReview:`n  ct setup --dry-run`nInstall:`n  ct setup"
        }
    } catch {
        Write-Host "ct remains installed. Onboarding $phase failed: $_"
        Show-CtNextSteps
    } finally {
        $env:CT_CONFIG_DIR = $previousConfigDir
    }
}

function Install-Ct {
    $ErrorActionPreference = 'Stop'
    if ($env:OS -ne 'Windows_NT') { throw 'This installer requires Windows.' }
    $arch = $env:PROCESSOR_ARCHITEW6432
    if (-not $arch) { $arch = $env:PROCESSOR_ARCHITECTURE }
    $asset = Get-CtAsset $arch
    if (-not $env:LOCALAPPDATA -or -not [IO.Path]::IsPathRooted($env:LOCALAPPDATA)) { throw 'LOCALAPPDATA must be an absolute path.' }
    $installDir = [IO.Path]::GetFullPath((Join-Path $env:LOCALAPPDATA 'Programs\ct'))
    $target = Join-Path $installDir 'ct.exe'
    Assert-CtPlainPath $target
    # Validate the owned entry before downloading or changing anything.
    $null = Add-CtPath '' $installDir
    $tempDir = Join-Path ([IO.Path]::GetTempPath()) ('ct-install-' + [Guid]::NewGuid().ToString('N'))
    $stage = $null
    $oldTls = [Net.ServicePointManager]::SecurityProtocol
    try {
        [Net.ServicePointManager]::SecurityProtocol = $oldTls -bor [Net.SecurityProtocolType]::Tls12
        $null = [IO.Directory]::CreateDirectory($tempDir)
        $download = Join-Path $tempDir $asset
        $checksums = Join-Path $tempDir 'checksums.txt'
        $base = 'https://github.com/GIRIBUIN/ct/releases/latest/download'
        Receive-CtFile "$base/$asset" $download
        Receive-CtFile "$base/checksums.txt" $checksums
        Assert-CtChecksum $download $checksums $asset
        $null = [IO.Directory]::CreateDirectory($installDir)
        Assert-CtPlainPath $target
        $stage = Join-Path $installDir ('.ct-' + [Guid]::NewGuid().ToString('N') + '.exe')
        [IO.File]::Copy($download, $stage, $false)
        & $stage --version | Out-Null
        if ($LASTEXITCODE -ne 0) { throw 'Downloaded ct failed its version check; existing binary preserved.' }
        if (Test-Path -LiteralPath $target) {
            # PowerShell 5.1 coerces $null to an empty string for this .NET method.
            [IO.File]::Replace($stage, $target, [NullString]::Value)
        } else {
            [IO.File]::Move($stage, $target)
        }
        $userPath = Get-CtUserPath
        $newPath = Add-CtPath $userPath $installDir
        if ($newPath -cne $userPath) { Set-CtUserPath $newPath }
        $env:PATH = Add-CtPath $env:PATH $installDir
        & $target --version
        if ($LASTEXITCODE -ne 0) { throw 'Installed ct failed its version check.' }
        Write-Host "Installed: $target"
        Start-CtOnboarding $target
        Write-Host 'Other terminals may need restarting. Rerun this installer to update ct.'
    } finally {
        [Net.ServicePointManager]::SecurityProtocol = $oldTls
        # Delete only the exact temporary files created by this invocation, never a tree.
        if ($stage -and [IO.File]::Exists($stage)) { [IO.File]::Delete($stage) }
        foreach ($name in @($asset, 'checksums.txt')) {
            $file = Join-Path $tempDir $name
            if ([IO.File]::Exists($file)) { [IO.File]::Delete($file) }
        }
        if ([IO.Directory]::Exists($tempDir)) { [IO.Directory]::Delete($tempDir, $false) }
    }
}

if ($MyInvocation.InvocationName -ne '.') { Install-Ct }
