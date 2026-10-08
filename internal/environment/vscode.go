package environment

import "strings"

func (c checker) checkExtensions(code, profile string, lookupErr error) []Check {
	var results []Check
	installed := make(map[string]bool)
	queryFailed := lookupErr != nil
	if !queryFailed {
		data, err := c.extensions(code, profile)
		if err != nil {
			queryFailed = true
			results = append(results, Check{"VS Code Extensions", "extension query", Fail, failureDetail(data, err)})
		} else {
			for _, line := range strings.Split(string(data), "\n") {
				installed[strings.ToLower(strings.TrimSpace(line))] = true
			}
		}
	}
	for _, extension := range []struct{ name, id string }{
		{"C/C++", "ms-vscode.cpptools"},
		{"CPH", "divyanshuagrawal.competitive-programming-helper"},
	} {
		result := Check{"VS Code Extensions", extension.name, OK, extension.id}
		if queryFailed {
			result.Status, result.Detail = Skip, extension.id+" (extension list unavailable)"
		} else if !installed[extension.id] {
			result.Status, result.Detail = Fail, extension.id+" (not installed)"
		}
		results = append(results, result)
	}
	return results
}
