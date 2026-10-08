package command

import (
	"bytes"
	"strings"
	"testing"

	"github.com/GIRIBUIN/ct/internal/version"
)

func TestVersionWithoutConfiguration(t *testing.T) {
	t.Setenv("CT_CONFIG_DIR", t.TempDir())
	previous := version.Version
	t.Cleanup(func() { version.Version = previous })
	for _, value := range []string{"dev", "v0.1.0"} {
		version.Version = value
		var output bytes.Buffer
		if err := Run([]string{"--version"}, strings.NewReader(""), &output); err != nil || output.String() != "ct "+value+"\n" {
			t.Fatalf("version output %q, error %v", output.String(), err)
		}
	}
	if _, err := parse([]string{"71A", "--version"}); err == nil {
		t.Fatal("version should be a standalone option")
	}
}
