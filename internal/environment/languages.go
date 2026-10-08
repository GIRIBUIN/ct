package environment

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func (c checker) languageChecks(selected string) []Check {
	checks := map[string]func() []Check{
		"cpp": c.cppChecks, "python": c.pythonChecks, "java": c.javaChecks, "rust": c.rustChecks,
	}
	if check, ok := checks[selected]; ok {
		return check()
	}
	return []Check{{"Environment", "provider", Skip, "no built-in environment provider; workspace generation is supported, language diagnostics are unavailable"}}
}

func (c checker) cppChecks() []Check {
	compiler, compilerErr := c.lookup("g++")
	debugger, debuggerErr := c.lookup("gdb")
	results := []Check{c.tool("g++", compiler, compilerErr), c.tool("GDB", debugger, debuggerErr)}
	if compilerErr != nil {
		return append(results, Check{"Capabilities", "C++20 compile", Skip, "g++ unavailable"}, Check{"Capabilities", "bits/stdc++.h", Skip, "g++ unavailable"})
	}
	return append(results, c.compile(compiler)...)
}

func (c checker) pythonChecks() []Check {
	for _, candidate := range []struct {
		name   string
		prefix []string
	}{
		{"python3", nil}, {"python", nil}, {"py", []string{"-3"}},
	} {
		path, err := c.lookup(candidate.name)
		if err != nil {
			continue
		}
		args := append(candidate.prefix, "-c", "import sys; print('CT_PYTHON', sys.version.split()[0], sys.executable); sys.exit(0 if sys.version_info.major == 3 else 1)")
		data, err := c.run(path, args...)
		if err == nil && strings.HasPrefix(strings.TrimSpace(string(data)), "CT_PYTHON ") {
			return []Check{{"Tools", "Python interpreter", OK, path + " — " + strings.TrimPrefix(concise(data), "CT_PYTHON ")}}
		}
	}
	return []Check{{"Tools", "Python interpreter", Fail, "No usable Python 3 interpreter. Install Python from https://www.python.org/downloads/ and enable its CLI; ct does not install Python."}}
}

var javaVersion = regexp.MustCompile(`(?:javac |(?:openjdk|java) (?:version ")?)([0-9]+)(?:\.([0-9]+))?`)

func (c checker) javaTool(name string) (string, Check) {
	path, err := c.lookup(name)
	if err != nil {
		return "", Check{"Tools", name, Fail, "not found; install a JDK 17+ (not just a JRE): " + err.Error()}
	}
	data, err := c.run(path, "-version")
	result := Check{"Tools", name, OK, path + " — " + concise(data)}
	if err != nil {
		result.Status, result.Detail = Fail, path+": "+failureDetail(data, err)
		return path, result
	}
	match := javaVersion.FindStringSubmatch(string(data))
	major := 0
	if len(match) > 0 {
		major, _ = strconv.Atoi(match[1])
		if major == 1 {
			major, _ = strconv.Atoi(match[2])
		}
	}
	if major < 17 {
		result.Status = Fail
		result.Detail += "; ct expects Java 17 or newer (version unrecognized or below 17)"
	}
	return path, result
}

func (c checker) javaChecks() []Check {
	_, java := c.javaTool("java")
	javac, compiler := c.javaTool("javac")
	results := []Check{java, compiler}
	if compiler.Status != OK {
		return append(results, Check{"Capabilities", "Java 17+ compile", Skip, "usable javac 17+ required"})
	}
	return append(results, c.probe("Java 17+ compile", "CTDoctor.java", "public class CTDoctor { public static void main(String[] args) {} }\n", func(dir, source string) ([]byte, error) {
		return c.run(javac, "--release", "17", "-d", dir, source)
	})...)
}

func (c checker) rustChecks() []Check {
	path, err := c.lookup("rustc")
	if err != nil {
		return []Check{{"Tools", "rustc", Fail, "not found; install Rust via https://rustup.rs/ and reopen your terminal"}, {"Capabilities", "Rust 2021 compile", Skip, "rustc unavailable"}}
	}
	data, err := c.run(path, "--version")
	tool := Check{"Tools", "rustc", OK, path + " — " + concise(data)}
	if err != nil || !strings.HasPrefix(strings.TrimSpace(string(data)), "rustc ") {
		if err == nil {
			err = fmt.Errorf("unrecognized rustc version")
		}
		tool.Status, tool.Detail = Fail, path+": "+failureDetail(data, err)+"; configure a stable toolchain using official rustup"
		return []Check{tool, {"Capabilities", "Rust 2021 compile", Skip, "usable rustc required"}}
	}
	return append([]Check{tool}, c.probe("Rust 2021 compile", "ct_doctor.rs", "fn main() {}\n", func(dir, source string) ([]byte, error) {
		return c.run(path, "--edition=2021", source, "-o", filepath.Join(dir, "ct-doctor.exe"))
	})...)
}

// Compile only inside an OS temporary directory; never execute the generated program.
func (c checker) probe(name, filename, source string, compile func(string, string) ([]byte, error)) (results []Check) {
	result := Check{"Capabilities", name, OK, ""}
	dir, err := os.MkdirTemp("", "ct-doctor-")
	if err != nil {
		return []Check{{"Capabilities", name, Fail, err.Error()}}
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			results = append(results, Check{"Capabilities", "temporary cleanup", Fail, err.Error()})
		}
	}()
	file := filepath.Join(dir, filename)
	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		result.Status, result.Detail = Fail, err.Error()
	} else if data, err := compile(dir, file); err != nil {
		result.Status, result.Detail = Fail, failureDetail(data, err)
	}
	return []Check{result}
}
