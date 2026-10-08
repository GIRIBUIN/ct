package command

import (
	"bytes"
	"strings"
	"testing"

	"github.com/GIRIBUIN/ct/internal/environment"
)

func TestDoctorParsing(t *testing.T) {
	for _, args := range [][]string{{"doctor"}, {"doctor", "--help"}, {"doctor", "-h"}} {
		opts, err := parse(args)
		if err != nil || !opts.doctor || opts.id != "" || opts.help != (len(args) > 1) {
			t.Fatalf("doctor parsed as problem: %+v, %v", opts, err)
		}
	}
	for _, args := range [][]string{{"doctor", "71A"}, {"doctor", "-l", "kotlin"}, {"doctor", "--unknown"}} {
		if _, err := parse(args); err == nil {
			t.Fatalf("invalid doctor arguments accepted: %v", args)
		}
	}
	var output bytes.Buffer
	if err := Run([]string{"--help"}, strings.NewReader(""), &output); err != nil || !strings.Contains(output.String(), "ct doctor") {
		t.Fatalf("help missing doctor: %q, %v", output.String(), err)
	}
}

func TestDoctorExitResult(t *testing.T) {
	for _, failed := range []bool{false, true} {
		checks := []environment.Check{{Section: "System", Name: "OS", Status: environment.OK, Detail: "test-os/test-arch"}}
		if failed {
			checks = append(checks,
				environment.Check{Section: "Tools", Name: "g++", Status: environment.Fail, Detail: "not found"},
				environment.Check{Section: "Capabilities", Name: "C++20 compile", Status: environment.Skip},
				environment.Check{Section: "Configuration", Name: "root", Status: environment.Fail, Detail: "not found"})
		}
		var output bytes.Buffer
		err := reportDoctor(&output, checks)
		if (err != nil) != failed {
			t.Fatalf("failure=%v returned %v", failed, err)
		}
		want := "Environment ready."
		if failed {
			want = "Environment has 2 issue(s)."
		}
		if !strings.Contains(output.String(), want) {
			t.Fatalf("incorrect summary: %s", output.String())
		}
	}
}

func TestDoctorDefaultLanguageLabel(t *testing.T) {
	checks := []environment.Check{
		{Section: "Language", Name: "selected", Status: environment.OK, Detail: "rust"},
		{Section: "Configuration", Name: "language", Status: environment.OK, Detail: "cpp"},
	}
	var output bytes.Buffer
	if err := reportDoctor(&output, checks); err != nil {
		t.Fatal(err)
	}
	text := strings.Join(strings.Fields(output.String()), " ")
	if !strings.Contains(text, "Language [OK] selected rust") ||
		!strings.Contains(text, "Configuration [OK] default language cpp") {
		t.Fatalf("unclear selected/default language output: %s", output.String())
	}
}
