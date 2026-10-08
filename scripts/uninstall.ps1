param([switch]$Purge)

function Get-CtRemovalPathKey([string]$Value) {
    $value = [Environment]::ExpandEnvironmentVariables($Value.Trim().Trim('"')).Replace('/', '\')
    if (-not $value) { return '' }
    try { $value = [IO.Path]::GetFullPath($value) } catch { }
    return $value.TrimEnd('\').ToLowerInvariant()
}

function Remove-CtPath([string]$Current, [string]$Entry) {
    $parts = @($Current.Split(';') | Where-Object { (Get-CtRemovalPathKey $_) -ne (Get-CtRemovalPathKey $Entry) })
    return $parts -join ';'
}

function Assert-CtRemovalPath([string]$Path) {
    $cursor = [IO.Path]::GetFullPath($Path)
    while ($cursor) {
        if (Test-Path -LiteralPath $cursor) {
            if ((Get-Item -LiteralPath $cursor -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) {
                throw "Refusing linked path: $cursor"
            }
        }
        $cursor = [IO.Path]::GetDirectoryName($cursor)
    }
}

function Get-CtCanonicalConfig([string]$AppData) {
    if (-not $AppData -or -not [IO.Path]::IsPathRooted($AppData)) { throw 'APPDATA must be an absolute path.' }
    return [IO.Path]::GetFullPath((Join-Path $AppData 'ct'))
}

function Remove-CtOwnedFile([string]$Directory, [string]$Name) {
    $file = Join-Path $Directory $Name
    Assert-CtRemovalPath $file
    if ([IO.File]::Exists($file)) { [IO.File]::Delete($file) }
    if ([IO.Directory]::Exists($Directory)) {
        if (@(Get-ChildItem -LiteralPath $Directory -Force).Count -eq 0) {
            [IO.Directory]::Delete($Directory, $false)
        } else {
            Write-Host "Preserved nonempty directory: $Directory"
        }
    }
}

function Get-CtRemovalUserPath { return [Environment]::GetEnvironmentVariable('Path', 'User') }
function Set-CtRemovalUserPath([string]$Value) { [Environment]::SetEnvironmentVariable('Path', $Value, 'User') }

function Uninstall-Ct([switch]$RemoveConfig) {
    $ErrorActionPreference = 'Stop'
    if ($env:OS -ne 'Windows_NT') { throw 'This uninstaller requires Windows.' }
    if (-not $env:LOCALAPPDATA -or -not [IO.Path]::IsPathRooted($env:LOCALAPPDATA)) { throw 'LOCALAPPDATA must be an absolute path.' }
    $installDir = [IO.Path]::GetFullPath((Join-Path $env:LOCALAPPDATA 'Programs\ct'))
    Remove-CtOwnedFile $installDir 'ct.exe'
    $current = Get-CtRemovalUserPath
    $updated = Remove-CtPath $current $installDir
    if ($updated -cne $current) { Set-CtRemovalUserPath $updated }
    $env:PATH = Remove-CtPath $env:PATH $installDir
    if ($RemoveConfig) {
        if ($env:CT_CONFIG_DIR) { Write-Host 'CT_CONFIG_DIR is not a purge target; only canonical config.json is eligible.' }
        # Never enumerate/delete arbitrary contents, even under the canonical directory.
        Remove-CtOwnedFile (Get-CtCanonicalConfig $env:APPDATA) 'config.json'
    }
    Write-Host 'ct removed. Solution files and development tools were preserved.'
    if (-not $RemoveConfig) { Write-Host 'ct configuration was preserved (use -Purge to remove canonical config.json).' }
}

if ($MyInvocation.InvocationName -ne '.') { Uninstall-Ct -RemoveConfig:$Purge }
