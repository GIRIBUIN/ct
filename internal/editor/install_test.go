package editor

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestInstallExtensionVerification(t *testing.T) {
	commandFailure := errors.New("CLI exited with status 1")
	queryFailure := errors.New("query failed")
	for _, tc := range []struct {
		name                 string
		installErr, queryErr error
		list                 string
		wantOK               bool
	}{
		{"normal", nil, nil, "other.extension\r\nredhat.java\r\n", true},
		{"normal but absent", nil, nil, "other.extension", false},
		{"command failure absent", commandFailure, nil, "", false},
		{"command failure present", commandFailure, nil, "redhat.java", true},
		{"wait delay present", exec.ErrWaitDelay, nil, "redhat.java", true},
		{"wait delay absent", exec.ErrWaitDelay, nil, "redhat.java-extra", false},
		{"query failure", nil, queryFailure, "redhat.java", false},
	} {
		for _, profile := range []string{"", "coding test"} {
			t.Run(tc.name+"/"+profile, func(t *testing.T) {
				var output bytes.Buffer
				ran, queried := false, false
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				run := func(cmd *exec.Cmd, _ io.Writer) error {
					ran = true
					want := append([]string{"code.exe"}, profileArgs(profile)...)
					want = append(want, "--install-extension", "redhat.java")
					if !reflect.DeepEqual(cmd.Args, want) {
						t.Fatalf("arguments = %q; want %q", cmd.Args, want)
					}
					cancel() // Verification must work even after install cancellation.
					return tc.installErr
				}
				query := func(ctx context.Context, path, gotProfile string) ([]byte, error) {
					queried = true
					if !ran || path != "code.exe" || gotProfile != profile || ctx.Err() != nil {
						t.Fatalf("invalid verification: ran=%v path=%q profile=%q err=%v", ran, path, gotProfile, ctx.Err())
					}
					return []byte(tc.list), tc.queryErr
				}
				err := installExtension(ctx, "code.exe", profile, "redhat.java", &output, run, query)
				if !queried || (err == nil) != tc.wantOK {
					t.Fatalf("queried=%v error=%v; want success=%v", queried, err, tc.wantOK)
				}
				if !tc.wantOK && tc.installErr != nil && !errors.Is(err, tc.installErr) {
					t.Fatalf("lost original command error: %v", err)
				}
				if tc.wantOK && tc.installErr != nil && !strings.Contains(output.String(), "verified redhat.java") {
					t.Fatalf("abnormal CLI result was not explained: %q", output.String())
				}
			})
		}
	}
}

func TestRunExtensionInstallNonInteractive(t *testing.T) {
	for _, direct := range []bool{false, true} {
		t.Run(fmt.Sprintf("direct=%v", direct), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestExtensionInstallHelper$")
			cmd.Env = append(os.Environ(), "CT_TEST_INSTALL_HELPER=1")
			cmd.WaitDelay = time.Second
			// Model the setup confirmation reader. The installer must discard it.
			cmd.Stdin = bufio.NewReader(strings.NewReader("unexpected terminal input\n"))
			var buffer bytes.Buffer
			var output io.Writer = &buffer
			if direct {
				file, err := os.CreateTemp(t.TempDir(), "output")
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				output = file
			}
			if err := runExtensionInstall(cmd, output); err != nil {
				t.Fatal(err)
			}
			if cmd.Stdin != nil {
				t.Fatal("installation inherited stdin")
			}
			if _, ok := cmd.Stdout.(*os.File); !ok {
				t.Fatal("stdout uses a Go copy pipe")
			}
			if _, ok := cmd.Stderr.(*os.File); !ok {
				t.Fatal("stderr uses a Go copy pipe")
			}
			if direct {
				file := output.(*os.File)
				if _, err := file.Seek(0, io.SeekStart); err != nil {
					t.Fatal(err)
				}
				if _, err := io.Copy(&buffer, file); err != nil {
					t.Fatal(err)
				}
			}
			for _, marker := range []string{"installed", "diagnostic"} {
				if !strings.Contains(buffer.String(), marker) {
					t.Fatalf("lost output: %q", buffer.String())
				}
			}
		})
	}
}

func TestExtensionInstallHelper(t *testing.T) {
	if os.Getenv("CT_TEST_INSTALL_HELPER") != "1" {
		return
	}
	var b [1]byte
	if n, err := os.Stdin.Read(b[:]); n != 0 || err != io.EOF {
		fmt.Fprintln(os.Stderr, "expected immediate EOF on stdin")
		os.Exit(2)
	}
	fmt.Fprintln(os.Stdout, "installed")
	fmt.Fprintln(os.Stderr, "diagnostic")
	os.Exit(0)
}
