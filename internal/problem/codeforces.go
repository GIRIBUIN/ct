package problem

import (
	"fmt"
	"regexp"
)

var codeforcesID = regexp.MustCompile(`^[0-9]+[A-Za-z][0-9]*$`)

func validateCodeforcesID(id string) error {
	if !codeforcesID.MatchString(id) {
		return fmt.Errorf("invalid Codeforces problem ID %q (example: 71A)", id)
	}
	return nil
}
