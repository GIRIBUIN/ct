package command

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/GIRIBUIN/ct/internal/environment"
	"github.com/GIRIBUIN/ct/internal/installer"
)

func TestSetupParsing(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want options
	}{
		{[]string{"setup"}, options{setup: true}},
		{[]string{"setup", "--yes"}, options{setup: true, yes: true}},
		{[]string{"setup", "--dry-run"}, options{setup: true, dryRun: true}},
		{[]string{"setup", "--yes", "--dry-run"}, options{setup: true, yes: true, dryRun: true}},
		{[]string{"setup", "--help"}, options{setup: true, help: true}},
	} {
		got, err := parse(tt.args)
		if err != nil || got != tt.want {
			t.Fatalf("parse %v = %+v, %v", tt.args, got, err)
		}
	}
	for _, args := range [][]string{{"setup", "71A"}, {"setup", "--force"}, {"setup", "-y"}, {"71A", "--yes"}, {"doctor", "--yes"}} {
		if _, err := parse(args); err == nil {
			t.Fatalf("unexpected flags accepted: %v", args)
		}
	}
}

func TestSetupReady(t *testing.T) {
	var output bytes.Buffer
	err := runSetup(options{}, strings.NewReader(""), &output, func() []environment.Check { return []environment.Check{{Status: environment.OK}} }, func([]environment.Check) (installer.Plan, error) {
		t.Fatal("ready environment planned installation")
		return installer.Plan{}, nil
	})
	if err != nil || !strings.Contains(output.String(), "Nothing to install.") {
		t.Fatalf("ready: %s, %v", output.String(), err)
	}
}

func TestSetupApprovalAndDryRun(t *testing.T) {
	for _, tt := range []struct {
		name, input   string
		opts          options
		execute, fail bool
	}{
		{"dry-run", "", options{dryRun: true}, false, false},
		{"dry-run-wins", "", options{dryRun: true, yes: true}, false, false},
		{"declined", "n\n", options{}, false, true},
		{"EOF", "", options{}, false, true},
		{"yes-flag", "", options{yes: true}, true, false},
		{"explicit-yes", "y\n", options{}, true, false},
		{"enter", "\n", options{}, true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			calls, diagnoses := 0, 0
			diagnose := func() []environment.Check {
				diagnoses++
				if diagnoses == 1 {
					return []environment.Check{{Name: "CPH", Status: environment.Fail}}
				}
				return []environment.Check{{Status: environment.OK}}
			}
			build := func([]environment.Check) (installer.Plan, error) {
				return installer.Plan{Actions: []installer.Action{{ID: "CPH", Description: "Install CPH", Details: []string{"exact command"}, Execute: func() error {
					if !strings.Contains(output.String(), "exact command") {
						t.Fatal("mutation before displayed plan")
					}
					calls++
					return nil
				}}}}, nil
			}
			err := runSetup(tt.opts, strings.NewReader(tt.input), &output, diagnose, build)
			if (err != nil) != tt.fail || (calls == 1) != tt.execute {
				t.Fatalf("calls=%d err=%v output=%s", calls, err, output.String())
			}
			if tt.execute && diagnoses != 2 {
				t.Fatal("final doctor not run")
			}
			if !tt.execute && diagnoses != 1 {
				t.Fatal("unexpected post-check")
			}
			if (tt.opts.dryRun || tt.opts.yes) && strings.Contains(output.String(), "Continue?") {
				t.Fatal("unexpected prompt")
			}
		})
	}
}

func TestSetupFailureContinuesIndependentActions(t *testing.T) {
	calls := []string{}
	diagnoses := 0
	var output bytes.Buffer
	err := runSetup(options{yes: true}, strings.NewReader(""), &output, func() []environment.Check {
		diagnoses++
		return []environment.Check{{Name: "g++", Status: environment.Fail}}
	}, func([]environment.Check) (installer.Plan, error) {
		return installer.Plan{Actions: []installer.Action{
			{ID: "msys", Description: "Install MSYS2", Execute: func() error { calls = append(calls, "msys"); return errors.New("failed") }},
			{ID: "gcc", Description: "Install GCC", DependsOn: []string{"msys"}, Execute: func() error { t.Fatal("dependency failure ignored"); return nil }},
			{ID: "CPH", Description: "Install CPH", Execute: func() error { calls = append(calls, "CPH"); return nil }},
		}}, nil
	})
	if err == nil || strings.Join(calls, ",") != "msys,CPH" || diagnoses != 2 || !strings.Contains(output.String(), "[SKIP]") {
		t.Fatalf("failure handling: %v,%v,%d,%s", err, calls, diagnoses, output.String())
	}
}

func TestSetupFinalDoctorFailure(t *testing.T) {
	diagnoses := 0
	err := runSetup(options{yes: true}, strings.NewReader(""), &bytes.Buffer{}, func() []environment.Check {
		diagnoses++
		return []environment.Check{{Name: "CPH", Status: environment.Fail}}
	}, func([]environment.Check) (installer.Plan, error) {
		return installer.Plan{Actions: []installer.Action{{ID: "CPH", Execute: func() error { return nil }}}}, nil
	})
	if err == nil || diagnoses != 2 {
		t.Fatalf("final failure ignored: %v (%d checks)", err, diagnoses)
	}
}

func TestSetupUnsupportedDryRunAndActual(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		var output bytes.Buffer
		err := runSetup(options{dryRun: dryRun}, strings.NewReader(""), &output, func() []environment.Check { return []environment.Check{{Name: "g++", Status: environment.Fail}} }, func([]environment.Check) (installer.Plan, error) {
			return installer.Plan{Notes: []string{"unsupported distro"}}, nil
		})
		if (err == nil) != dryRun || !strings.Contains(output.String(), "unsupported distro") || strings.Contains(output.String(), "Continue?") {
			t.Fatalf("unsupported result: %v, %s", err, output.String())
		}
	}
}
