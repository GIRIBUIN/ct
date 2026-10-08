package scripts

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func readWorkflow(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", ".github", "workflows", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestReleaseTriggerVersionTagsOnly(t *testing.T) {
	workflow := readWorkflow(t, "release.yml")
	// Check the complete trigger block rather than a tag filter appearing in
	// an unrelated job. Ignore whitespace, quoting and list presentation.
	block := regexp.MustCompile(`(?m)^on:\s*\r?\n((?:[ \t]+[^\n]*\n)+)`).FindStringSubmatch(workflow)
	if len(block) != 2 {
		t.Fatal("release trigger block not found")
	}
	trigger := regexp.MustCompile(`(?m)^[ \t]*-[ \t]+`).ReplaceAllString(block[1], "")
	trigger = strings.NewReplacer("'", "", `"`, "", "[", " ", "]", " ", ",", " ").Replace(trigger)
	if got := strings.Join(strings.Fields(trigger), " "); got != "push: tags: v*" {
		t.Fatalf("release must run only on version-tag pushes: %s", got)
	}
}

func TestWorkflowUbuntuRunnersPinned(t *testing.T) {
	for _, name := range []string{"release.yml", "ci.yml"} {
		t.Run(name, func(t *testing.T) {
			workflow := readWorkflow(t, name)
			runners := regexp.MustCompile(`\bubuntu-[A-Za-z0-9.]+\b`).FindAllString(workflow, -1)
			if len(runners) == 0 {
				t.Fatal("Linux runner not found")
			}
			for _, runner := range runners {
				if runner != "ubuntu-24.04" {
					t.Fatalf("unexpected Ubuntu runner: %s", runner)
				}
			}
		})
	}
}
