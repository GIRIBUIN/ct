package environment

import "strings"

// Version strings are informational: presence and compilation decide readiness.
func (c checker) tool(name, path string, lookupErr error) Check {
	if lookupErr != nil {
		return Check{"Tools", name, Fail, "not found: " + lookupErr.Error()}
	}
	data, err := c.run(path, "--version")
	detail := path
	if err != nil {
		detail += " (version unavailable: " + failureDetail(data, err) + ")"
	} else if line := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)[0]; line != "" {
		detail += " — " + concise([]byte(line))
	} else {
		detail += " (version unavailable)"
	}
	return Check{"Tools", name, OK, detail}
}
