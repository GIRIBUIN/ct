package command

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/GIRIBUIN/ct/internal/environment"
	"github.com/GIRIBUIN/ct/internal/installer"
)

func setup(opts options, input io.Reader, output io.Writer) error {
	reader := bufio.NewReader(input)
	return runSetup(opts, reader, output, func() []environment.Check { return environment.Diagnose(opts.language) }, func(checks []environment.Check) (installer.Plan, error) {
		return installer.BuildPlan(checks, reader, output)
	})
}

func runSetup(opts options, input io.Reader, output io.Writer, diagnose func() []environment.Check, build func([]environment.Check) (installer.Plan, error)) error {
	fmt.Fprintln(output, "Checking environment...")
	checks := diagnose()
	for _, check := range checks {
		if check.Section == "Language" {
			fmt.Fprintf(output, "Selected language: %s\n", check.Detail)
		}
	}
	missing := false
	for _, check := range checks {
		if check.Status == environment.Fail {
			missing = true
			break
		}
	}
	if !missing {
		fmt.Fprintln(output, "Environment is already ready.\nNothing to install.")
		return nil
	}
	fmt.Fprintln(output, "\nMissing components / failed checks:")
	for _, check := range checks {
		if check.Status == environment.Fail {
			fmt.Fprintf(output, "  - %s: %s\n", check.Name, check.Detail)
		}
	}
	plan, err := build(checks)
	if err != nil {
		return fmt.Errorf("plan setup: %w", err)
	}
	fmt.Fprintln(output, "\nPlanned actions:")
	for i, action := range plan.Actions {
		fmt.Fprintf(output, "  %d. %s\n", i+1, action.Description)
		for _, detail := range action.Details {
			fmt.Fprintf(output, "     %s\n", detail)
		}
	}
	if len(plan.Actions) == 0 {
		fmt.Fprintln(output, "  No supported automatic actions.")
	}
	for _, note := range plan.Notes {
		fmt.Fprintln(output, "Note:", note)
	}
	if opts.dryRun {
		fmt.Fprintln(output, "\nDry run: no changes made.")
		return nil
	}
	if len(plan.Actions) == 0 {
		return reportDoctor(output, checks)
	}
	if !opts.yes {
		reader, ok := input.(*bufio.Reader)
		if !ok {
			reader = bufio.NewReader(input)
		}
		for {
			fmt.Fprint(output, "\nContinue? [Y/n]: ")
			answer, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("setup approval not received: %w", err)
			}
			switch strings.ToLower(strings.TrimSpace(answer)) {
			case "", "y", "yes":
			case "n", "no":
				fmt.Fprintln(output, "Setup cancelled. No changes made.")
				return errors.New("setup cancelled; environment is not ready")
			default:
				fmt.Fprintln(output, "Please answer y or n.")
				continue
			}
			break
		}
	}
	fmt.Fprintln(output, "\nSetup results")
	succeeded := make(map[string]bool)
	actionFailed := false
	for _, action := range plan.Actions {
		blocked := false
		for _, dependency := range action.DependsOn {
			if !succeeded[dependency] {
				blocked = true
				break
			}
		}
		if blocked {
			fmt.Fprintf(output, "[SKIP] %s (dependency failed)\n", action.Description)
			continue
		}
		if err := action.Execute(); err != nil {
			actionFailed = true
			fmt.Fprintf(output, "[FAIL] %s: %v\n", action.Description, err)
		} else {
			succeeded[action.ID] = true
			fmt.Fprintf(output, "[OK] %s\n", action.Description)
		}
	}
	fmt.Fprintln(output, "\nRunning doctor...")
	if err := reportDoctor(output, diagnose()); err != nil {
		return err
	}
	if actionFailed {
		return errors.New("one or more setup actions failed; review setup results")
	}
	return nil
}
