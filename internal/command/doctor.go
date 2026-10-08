package command

import (
	"fmt"
	"io"
	"strings"

	"github.com/GIRIBUIN/ct/internal/environment"
)

func doctor(output io.Writer, language string) error {
	return reportDoctor(output, environment.Diagnose(language))
}

func reportDoctor(output io.Writer, checks []environment.Check) error {
	var report strings.Builder
	fmt.Fprintln(&report, "Coding Test Environment")
	section, issues := "", 0
	for _, check := range checks {
		if check.Section != section {
			section = check.Section
			fmt.Fprintf(&report, "\n%s\n", section)
		}
		fmt.Fprintf(&report, "[%s] %-16s %s\n", check.Status, check.Name, check.Detail)
		if check.Status == environment.Fail {
			issues++
		}
	}
	if issues == 0 {
		fmt.Fprintln(&report, "\nEnvironment ready.")
	} else {
		fmt.Fprintf(&report, "\nEnvironment has %d issue(s).\n", issues)
	}
	if _, err := io.WriteString(output, report.String()); err != nil {
		return err
	}
	if issues > 0 {
		return fmt.Errorf("doctor found %d issue(s)", issues)
	}
	return nil
}
