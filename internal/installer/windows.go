package installer

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

const utf8Output = `$OutputEncoding = [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false); `
const readUserPath = `[Console]::Write([Environment]::GetEnvironmentVariable('Path', 'User'))`
const writeUserPath = `[Environment]::SetEnvironmentVariable('Path', $env:CT_SETUP_USER_PATH, 'User')`
const msysRegistry = `$locations = @(Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*','HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*','HKLM:\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*' -ErrorAction SilentlyContinue | Where-Object { $_.DisplayName -match '^MSYS2' } | ForEach-Object { if ($_.InstallLocation) { $_.InstallLocation } elseif ($_.UninstallString -match '^"?(.+?\.exe)') { Split-Path -LiteralPath $Matches[1] } }); ConvertTo-Json -InputObject $locations -Compress`

func powershell(path, script string) invocation {
	return invocation{path: path, args: []string{"-NoProfile", "-NonInteractive", "-Command", utf8Output + script}}
}

// AppendWindowsPath preserves existing bytes and compares entries using Windows
// case, separator, surrounding quote, dot-segment and trailing-slash conventions.
func AppendWindowsPath(existing, entry string) (string, bool, error) {
	entry = strings.Trim(strings.TrimSpace(entry), `"`)
	if entry == "" || strings.ContainsAny(entry, ";\"\r\n\x00") {
		return "", false, fmt.Errorf("unsafe PATH entry %q", entry)
	}
	for _, current := range strings.Split(existing, ";") {
		if normalizedWindowsPath(current) == normalizedWindowsPath(entry) {
			return existing, false, nil
		}
	}
	if existing == "" {
		return entry, true, nil
	}
	separator := ";"
	if strings.HasSuffix(existing, ";") {
		separator = ""
	}
	return existing + separator + entry, true, nil
}

func normalizedWindowsPath(value string) string {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	return strings.ToLower(strings.TrimRight(path.Clean(strings.ReplaceAll(value, `\`, "/")), "/"))
}

func (h host) discoverMSYS(ps string) (string, bool, error) {
	candidates := []string{}
	for _, name := range []string{"MSYS2_ROOT", "MSYSTEM_PREFIX"} {
		if value := h.getenv(name); value != "" {
			candidates = append(candidates, value)
		}
	}
	for _, name := range []string{"pacman.exe", "bash.exe", "g++.exe", "gdb.exe"} {
		if value, err := h.lookup(name); err == nil {
			candidates = append(candidates, filepath.Dir(value))
		}
	}
	for _, entry := range strings.Split(h.getenv("PATH"), ";") {
		if entry != "" {
			candidates = append(candidates, strings.Trim(entry, `"`))
		}
	}
	if drive := h.getenv("SystemDrive"); drive != "" {
		candidates = append(candidates, filepath.Join(drive+string(filepath.Separator), "msys64"))
	}
	if local := h.getenv("LOCALAPPDATA"); local != "" {
		candidates = append(candidates, filepath.Join(local, "Programs", "ct-msys2"), filepath.Join(local, "Programs", "MSYS2"))
	}
	suitable := func(root string) bool {
		return h.exists(filepath.Join(root, "usr", "bin", "pacman.exe")) && h.exists(filepath.Join(root, "usr", "bin", "bash.exe"))
	}
	for _, candidate := range candidates {
		for i := 0; i < 4; i++ {
			if suitable(candidate) {
				return candidate, true, nil
			}
			parent := filepath.Dir(candidate)
			if candidate == parent {
				break
			}
			candidate = parent
		}
	}
	// Consult registered installations before proposing a new one.
	data, err := h.query(powershell(ps, msysRegistry))
	if err != nil {
		return "", false, fmt.Errorf("cannot inspect existing MSYS2 registrations: %w", err)
	}
	var registered []string
	if err := json.Unmarshal(data, &registered); err != nil {
		return "", false, fmt.Errorf("decode MSYS2 registrations: %w", err)
	}
	for _, root := range registered {
		if suitable(root) {
			return root, true, nil
		}
	}
	if len(registered) > 0 {
		return "", true, nil
	}
	return "", false, nil
}

func (h host) planWindows(plan *Plan, compiler, debugger bool) {
	if h.arch != "amd64" {
		plan.Notes = append(plan.Notes, "Automatic Windows toolchain installation supports amd64 UCRT64 only.")
		return
	}
	ps, err := h.lookup("powershell.exe")
	if err != nil {
		plan.Notes = append(plan.Notes, "PowerShell is unavailable; MSYS2 discovery/User PATH configuration needs manual attention.")
		return
	}
	root, found, err := h.discoverMSYS(ps)
	if err != nil {
		plan.Notes = append(plan.Notes, err.Error()+"; no new MSYS2 installation will be attempted.")
		return
	}
	if found && root == "" {
		plan.Notes = append(plan.Notes, "An MSYS2 installation is registered but unusable. Repair it manually; setup will not install a second copy.")
		return
	}
	dependencies := []string{}
	if !found {
		winget, err := h.lookup("winget.exe")
		if err != nil {
			plan.Notes = append(plan.Notes, "No suitable MSYS2 or winget found. Install MSYS2 from https://www.msys2.org/ and retry.")
			return
		}
		local := h.getenv("LOCALAPPDATA")
		if local == "" {
			plan.Notes = append(plan.Notes, "LOCALAPPDATA is unavailable; cannot choose a per-user MSYS2 location.")
			return
		}
		root = filepath.Join(local, "Programs", "ct-msys2")
		if h.exists(root) {
			plan.Notes = append(plan.Notes, "MSYS2 target already exists but is unusable: "+root+". Repair it manually.")
			return
		}
		if strings.ContainsAny(root, " \t\r\n") {
			plan.Notes = append(plan.Notes, "MSYS2 needs a path without whitespace. Install it manually in a suitable directory and rerun setup.")
			return
		}
		c := invocation{path: winget, args: []string{"install", "--id", "MSYS2.MSYS2", "--exact", "--source", "winget", "--scope", "user", "--location", root, "--silent", "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}}
		plan.Actions = append(plan.Actions, h.commandAction("msys2", "Install MSYS2 for the current user via winget into "+root+" (accept source/package agreements)", c))
		dependencies = []string{"msys2"}
	}
	bin := filepath.Join(root, "ucrt64", "bin")
	packages := []string{}
	if compiler && !h.exists(filepath.Join(bin, "g++.exe")) {
		packages = append(packages, "mingw-w64-ucrt-x86_64-gcc")
	}
	if debugger && !h.exists(filepath.Join(bin, "gdb.exe")) {
		packages = append(packages, "mingw-w64-ucrt-x86_64-gdb")
	}
	if len(packages) > 0 {
		c := invocation{path: filepath.Join(root, "usr", "bin", "bash.exe"), args: []string{"--login", "-c", "exec pacman -S --needed --noconfirm " + strings.Join(packages, " ")}, env: []string{"MSYSTEM=UCRT64", "CHERE_INVOKING=1"}}
		plan.Actions = append(plan.Actions, h.commandAction("ucrt-packages", "Install missing UCRT64 packages and required dependencies in "+root, c, dependencies...))
		dependencies = []string{"ucrt-packages"}
		plan.Notes = append(plan.Notes, "MSYS2 uses its existing package databases. If they are stale, update MSYS2 following its documentation and retry; setup does not perform a full system upgrade.")
	}
	userPath, err := h.query(powershell(ps, readUserPath))
	if err != nil {
		plan.Notes = append(plan.Notes, "Cannot read User PATH; configure "+bin+" manually: "+err.Error())
		return
	}
	_, userChange, err := AppendWindowsPath(string(userPath), bin)
	if err != nil {
		plan.Notes = append(plan.Notes, err.Error())
		return
	}
	_, processChange, err := AppendWindowsPath(h.getenv("PATH"), bin)
	if err != nil {
		plan.Notes = append(plan.Notes, err.Error())
		return
	}
	if userChange || processChange {
		description := "Refresh ct process PATH with " + bin + " for final doctor"
		details := []string{"Append the unquoted entry to the current process PATH; preserve all existing entries."}
		if userChange {
			description = "Append " + bin + " to User PATH (never Machine PATH) and refresh ct process PATH"
			details = append([]string{"[Environment]::SetEnvironmentVariable('Path', <current User PATH plus unquoted " + bin + ">, 'User'); re-read and deduplicate before writing. Restart other terminals to inherit the change."}, details...)
		}
		plan.Actions = append(plan.Actions, Action{ID: "ucrt-path", Description: description, Details: details, DependsOn: dependencies, Execute: func() error {
			if userChange {
				latest, err := h.query(powershell(ps, readUserPath))
				if err != nil {
					return err
				}
				updated, changed, err := AppendWindowsPath(string(latest), bin)
				if err != nil {
					return err
				}
				if changed {
					c := powershell(ps, writeUserPath)
					c.env = []string{"CT_SETUP_USER_PATH=" + updated}
					if err := h.run(c); err != nil {
						return err
					}
				}
			}
			updated, _, err := AppendWindowsPath(h.getenv("PATH"), bin)
			if err != nil {
				return err
			}
			return h.setenv("PATH", updated)
		}})
	}
}
