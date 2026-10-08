package installer

import (
	"fmt"
	"strings"
)

func (h host) planJava(plan *Plan) {
	switch h.goos {
	case "linux":
		h.planAPT(plan, []string{"openjdk-21-jdk"})
		plan.Notes = append(plan.Notes, "openjdk-21-jdk must be available in your configured apt repositories. If unavailable, install a trusted JDK 17+ manually; ct will not add repositories.")
	case "windows":
		winget, err := h.lookup("winget.exe")
		if err != nil {
			plan.Notes = append(plan.Notes, "winget is unavailable. Install a JDK 17+ manually from https://learn.microsoft.com/java/openjdk/download and enable java/javac on PATH.")
			return
		}
		ps, err := h.lookup("powershell.exe")
		if err != nil {
			plan.Notes = append(plan.Notes, "PowerShell is unavailable; install a JDK 17+ manually and reopen the terminal.")
			return
		}
		command := invocation{path: winget, args: []string{"install", "--id", "Microsoft.OpenJDK.21", "--exact", "--source", "winget", "--silent", "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}}
		plan.Actions = append(plan.Actions, h.commandAction("java-jdk", "Install Microsoft OpenJDK 21 LTS via winget (accept source/package agreements; installer may request elevation and register its PATH/environment)", command))
		plan.Actions = append(plan.Actions, Action{ID: "java-path", Description: "Refresh ct process PATH from registered Machine/User PATH for final Java diagnosis", Details: []string{"Read the installer's registered PATH; preserve other process entries. ct does not write persistent PATH or JAVA_HOME."}, DependsOn: []string{"java-jdk"}, Execute: func() error {
			data, err := h.query(powershell(ps, `[Console]::Write([Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('Path', 'User'))`))
			if err != nil {
				return fmt.Errorf("refresh JDK PATH: %w", err)
			}
			updated := string(data)
			for _, entry := range strings.Split(h.getenv("PATH"), ";") {
				if entry == "" {
					continue
				}
				updated, _, err = AppendWindowsPath(updated, entry)
				if err != nil {
					return err
				}
			}
			return h.setenv("PATH", updated)
		}})
		plan.Notes = append(plan.Notes, "If another Java installation still takes precedence on PATH, select the new JDK manually and rerun ct doctor -l java. ct will not remove or rewrite an existing Java installation.")
	default:
		plan.Notes = append(plan.Notes, "Automatic JDK installation is unsupported on "+h.goos+"; install a trusted JDK 17+ manually.")
	}
}
