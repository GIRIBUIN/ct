package editor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// InstallExtension runs an approved, non-interactive installation and checks
// its actual result in the same profile, including after an abnormal CLI exit.
func InstallExtension(ctx context.Context, path, profile, id string, output io.Writer) error {
	return installExtension(ctx, path, profile, id, output, runExtensionInstall, ListExtensions)
}

func installExtension(ctx context.Context, path, profile, id string, output io.Writer,
	run func(*exec.Cmd, io.Writer) error,
	query func(context.Context, string, string) ([]byte, error),
) error {
	cmd, err := InstallExtensionCommand(ctx, path, profile, id)
	if err != nil {
		return err
	}
	installErr := run(cmd, output)
	// Installation may have exhausted its deadline. Verification gets its own
	// bounded context so a timed-out command cannot prevent checking actual state.
	verifyCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	data, queryErr := query(verifyCtx, path, profile)
	if queryErr != nil {
		return errors.Join(installErr, fmt.Errorf("verify extension %s in VS Code profile %q: %w: %s", id, profile, queryErr, strings.TrimSpace(string(data))))
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.EqualFold(strings.TrimSpace(line), id) {
			if installErr != nil {
				fmt.Fprintf(output, "VS Code CLI returned %v; verified %s is installed in the selected profile.\n", installErr, id)
			}
			return nil
		}
	}
	return errors.Join(installErr, fmt.Errorf("extension %s is absent from VS Code profile %q after installation", id, profile))
}

func runExtensionInstall(cmd *exec.Cmd, output io.Writer) error {
	// nil stdin provides the null device. In particular, never copy from the
	// setup confirmation reader: a console Read can outlive the child process
	// and cannot be interrupted by closing os/exec's stdin pipe on WaitDelay.
	cmd.Stdin = nil
	if file, ok := output.(*os.File); ok {
		// Direct handles stream output without os/exec copy goroutines or pipes
		// that detached VS Code descendants could keep open.
		cmd.Stdout, cmd.Stderr = file, file
		return cmd.Run()
	}
	// Embedded callers and tests may supply arbitrary writers. Spool to a file
	// to retain the same process lifecycle; forward output after the CLI exits.
	file, err := os.CreateTemp("", "ct-extension-output-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	cmd.Stdout, cmd.Stderr = file, file
	runErr := cmd.Run()
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return errors.Join(runErr, err)
	}
	_, copyErr := io.Copy(output, file)
	return errors.Join(runErr, copyErr)
}
