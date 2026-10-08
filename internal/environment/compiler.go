package environment

import (
	"os"
	"path/filepath"
)

func (c checker) compile(compiler string) (results []Check) {
	dir, err := os.MkdirTemp("", "ct-doctor-")
	if err != nil {
		return []Check{{"Capabilities", "temporary directory", Fail, err.Error()}}
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			results = append(results, Check{"Capabilities", "temporary cleanup", Fail, err.Error()})
		}
	}()
	for _, probe := range []struct{ name, file, source string }{
		{"C++20 compile", "cpp20", "#include <concepts>\nstatic_assert(std::integral<int>);\nint main() { return 0; }\n"},
		{"bits/stdc++.h", "bits", "#include <bits/stdc++.h>\nint main() { std::vector<int> v{1}; return v[0] - 1; }\n"},
	} {
		result := Check{"Capabilities", probe.name, OK, ""}
		source := filepath.Join(dir, probe.file+".cpp")
		if err := os.WriteFile(source, []byte(probe.source), 0600); err != nil {
			result.Status, result.Detail = Fail, err.Error()
		} else if data, err := c.run(compiler, "-std=c++20", source, "-o", filepath.Join(dir, probe.file+".exe")); err != nil {
			result.Status, result.Detail = Fail, failureDetail(data, err)
		}
		results = append(results, result)
	}
	return results
}
