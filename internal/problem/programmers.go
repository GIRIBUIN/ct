package problem

import (
	"fmt"
	"regexp"
)

var programmersID = regexp.MustCompile(`^[0-9]+$`)

func programmersFilename(id string) (string, error) {
	if !programmersID.MatchString(id) {
		return "", fmt.Errorf("invalid Programmers problem ID %q (example: 181188)", id)
	}
	return "solution", nil
}
