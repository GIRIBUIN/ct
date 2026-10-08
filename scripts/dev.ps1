$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path

$tempCtConfig = Join-Path $repoRoot ".tmp-ct-config"
$tempCodingTest = Join-Path $repoRoot ".tmp-coding-test"
$exe = Join-Path $repoRoot "ct.exe"

$reset = $args -contains "--reset"
$clean = $args -contains "--clean"

$ctArgs = @(
    $args | Where-Object {
        $_ -ne "--reset" -and $_ -ne "--clean"
    }
)

function Remove-DevFiles {
    foreach ($target in @($tempCtConfig, $tempCodingTest, $exe)) {
        $resolvedTarget = [System.IO.Path]::GetFullPath($target)
        if ([System.IO.Path]::GetDirectoryName($resolvedTarget) -ne $repoRoot) {
            throw "Development cleanup target is outside the repository: $resolvedTarget"
        }
        if (Test-Path -LiteralPath $resolvedTarget) {
            Remove-Item -LiteralPath $resolvedTarget -Recurse -Force
        }
    }
}

if ($clean) {
    Remove-DevFiles
    Write-Host "Cleaned development environment."
    exit 0
}

if ($reset) {
    Remove-DevFiles
    Write-Host "Reset development environment."
}

if ($ctArgs.Count -eq 0) {
    Write-Host "Usage:"
    Write-Host "  .\scripts\dev.ps1 71A"
    Write-Host "  .\scripts\dev.ps1 158A -l py"
    Write-Host "  .\scripts\dev.ps1 181188 -p pg"
    Write-Host "  .\scripts\dev.ps1 --reset 71A"
    Write-Host "  .\scripts\dev.ps1 --clean"
    exit 0
}

Push-Location $repoRoot

$oldCtConfigDir = [Environment]::GetEnvironmentVariable('CT_CONFIG_DIR', 'Process')

try {
    $firstRun = -not (Test-Path -LiteralPath (Join-Path $tempCtConfig "config.json"))

    # Build latest ct
    go build -o $exe ./cmd/ct

    if ($LASTEXITCODE -ne 0) {
        throw "ct build failed."
    }

    # Test root
    New-Item -ItemType Directory -Force $tempCodingTest | Out-Null

    $env:CT_CONFIG_DIR = $tempCtConfig

    if ($firstRun) {
        # Avoid PowerShell's native pipeline input handling: after its EOF it
        # can leave a later native command disconnected from console stdin.
        # Only this separate process owns the synthetic bootstrap input pipe.
        $bootstrap = New-Object Diagnostics.Process
        $bootstrap.StartInfo.FileName = $exe
        $bootstrap.StartInfo.Arguments = 'config'
        $bootstrap.StartInfo.UseShellExecute = $false
        $bootstrap.StartInfo.CreateNoWindow = $true
        $bootstrap.StartInfo.RedirectStandardInput = $true
        try {
            $null = $bootstrap.Start()
            $answers = [Text.Encoding]::UTF8.GetBytes("$tempCodingTest`n`n`n`n")
            $bootstrap.StandardInput.BaseStream.Write($answers, 0, $answers.Length)
            $bootstrap.StandardInput.Close()
            $bootstrap.WaitForExit()
            $bootstrapExit = $bootstrap.ExitCode
        }
        finally {
            $bootstrap.Dispose()
        }
        if ($bootstrapExit -ne 0) {
            exit $bootstrapExit
        }
    }

    # No pipeline/redirection: inherit the original terminal stdin normally.
    & $exe @ctArgs

    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}
finally {
    [Environment]::SetEnvironmentVariable('CT_CONFIG_DIR', $oldCtConfigDir, 'Process')
    Pop-Location
}
