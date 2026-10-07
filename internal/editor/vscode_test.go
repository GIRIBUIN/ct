package editor

import (
	"reflect"
	"testing"
)

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
	cmd := launchCommand("code.exe", "root with spaces", "root with spaces/main.cpp")
	want := []string{"code.exe", "--reuse-window", "root with spaces", "--goto", "root with spaces/main.cpp"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("arguments = %q; want %q", cmd.Args, want)
	}
}
