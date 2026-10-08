package environment

import "strings"

var RequiredExtensions = []struct{ Name, ID string }{
	{"C/C++", "ms-vscode.cpptools"},
	{"CPH", "divyanshuagrawal.competitive-programming-helper"},
}

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
	for _, extension := range RequiredExtensions {
		result := Check{"VS Code Extensions", extension.Name, OK, extension.ID}
		if queryFailed {
			result.Status, result.Detail = Skip, extension.ID+" (extension list unavailable)"
		} else if !installed[extension.ID] {
			result.Status, result.Detail = Fail, extension.ID+" (not installed)"
		}
		results = append(results, result)
	}
	return results
}
