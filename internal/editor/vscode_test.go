package editor

import (
	"context"
	"reflect"
	"testing"
)

func TestExtensionCommandArguments(t *testing.T) {
	for _, profile := range []string{"", "my coding profile"} {
		cmd := extensionCommand(context.Background(), "code.exe", profile)
		want := []string{"code.exe"}
		if profile != "" {
			want = append(want, "--profile", profile)
		}
		want = append(want, "--list-extensions")
		if !reflect.DeepEqual(cmd.Args, want) {
			t.Fatalf("unexpected extension query: %q; want %q", cmd.Args, want)
		}
	}
}

func TestExtensionInstallArguments(t *testing.T) {
	for _, profile := range []string{"", "my coding profile"} {
		cmd, err := InstallExtensionCommand(context.Background(), "code.exe", profile, "ms-vscode.cpptools")
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"code.exe"}
		if profile != "" {
			want = append(want, "--profile", profile)
		}
		want = append(want, "--install-extension", "ms-vscode.cpptools")
		if !reflect.DeepEqual(cmd.Args, want) {
			t.Fatalf("install arguments = %q; want %q", cmd.Args, want)
		}
	}
}

func TestEditorEnvironment(t *testing.T) {
	environment := []string{
		"APPDATA=normal-user-data", "LOCALAPPDATA=normal-local-data",
		"HOME=normal-home", "XDG_CONFIG_HOME=normal-config", "PATH=normal-path",
		"CT_EDITOR_ROOT=coding-test", "CT_CONFIG_DIR=temporary-config",
		"ct_config_dir=temporary-config",
	}
	got := editorEnvironment(environment)
	want := environment[:6]
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("editor environment = %q; want %q", got, want)
	}
	if len(environment) != 8 || environment[6] != "CT_CONFIG_DIR=temporary-config" {
		t.Fatal("parent environment modified")
	}
}

func TestLaunchArguments(t *testing.T) {
	for _, profile := range []string{"", "my coding profile"} {
		cmd := launchCommand("code.exe", "root with spaces", "root with spaces/main.cpp", profile)
		want := []string{"code.exe"}
		if profile != "" {
			want = append(want, "--profile", profile)
		}
		want = append(want, "--reuse-window", "root with spaces", "root with spaces/main.cpp")
		if !reflect.DeepEqual(cmd.Args, want) {
			t.Fatalf("arguments = %q; want %q", cmd.Args, want)
		}
	}
}
