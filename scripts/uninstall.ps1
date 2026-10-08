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

function Remove-CtTemplates([string]$ConfigDirectory) {
    $base = [IO.Path]::GetFullPath((Join-Path $ConfigDirectory 'templates'))
    Assert-CtRemovalPath $base
    if (-not [IO.Directory]::Exists($base)) { return }
    foreach ($kind in @('languages', 'platforms')) {
        $group = Join-Path $base $kind
        Assert-CtRemovalPath $group
        if (-not [IO.Directory]::Exists($group)) { continue }
        foreach ($owner in @(Get-ChildItem -LiteralPath $group -Force)) {
            if ($owner.Name -cnotmatch '^[a-z][a-z0-9_-]*$') { continue }
            Assert-CtRemovalPath $owner.FullName
            if (-not $owner.PSIsContainer) { continue }
            foreach ($file in @(Get-ChildItem -LiteralPath $owner.FullName -Force)) {
                if ($file.Name -cnotmatch '^[a-z][a-z0-9_-]*\.tmpl$') { continue }
                Assert-CtRemovalPath $file.FullName
                if (-not $file.PSIsContainer) { [IO.File]::Delete($file.FullName) }
            }
            if (@(Get-ChildItem -LiteralPath $owner.FullName -Force).Count -eq 0) { [IO.Directory]::Delete($owner.FullName, $false) }
        }
        if (@(Get-ChildItem -LiteralPath $group -Force).Count -eq 0) { [IO.Directory]::Delete($group, $false) }
    }
    if (@(Get-ChildItem -LiteralPath $base -Force).Count -eq 0) { [IO.Directory]::Delete($base, $false) }
}

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
        if ($env:CT_CONFIG_DIR) { Write-Host 'CT_CONFIG_DIR is not a purge target; only canonical ct configuration/registry/templates are eligible.' }
        $canonical = Get-CtCanonicalConfig $env:APPDATA
        Remove-CtTemplates $canonical
        Remove-CtOwnedFile $canonical 'registry.json'
        Remove-CtOwnedFile $canonical 'config.json'
    }
    Write-Host 'ct removed. Solution files and development tools were preserved.'
    if (-not $RemoveConfig) { Write-Host 'ct configuration, registry and templates were preserved (use -Purge to remove canonical ct data).' }
}

if ($MyInvocation.InvocationName -ne '.') { Uninstall-Ct -RemoveConfig:$Purge }
