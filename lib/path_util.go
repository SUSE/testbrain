package lib

import (
	"path/filepath"
	"strings"
)

// CommonPathPrefix returns the common prefix of all the paths given.  This is
// a directory that contains all of the given paths.
func CommonPathPrefix(paths []string) (string, error) {
	if len(paths) == 0 {
		// No paths given!?
		return "", nil
	}
	pathParts := make([][]string, 0, len(paths))
	minLen := int((^uint(0)) >> 1) // max int
	for _, path := range paths {
		path, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		parts := strings.Split(path, string(filepath.Separator))
		pathParts = append(pathParts, parts)
		if minLen > len(parts) {
			minLen = len(parts)
		}
	}

	var matchingParts []string
	if '/' == filepath.Separator {
		// On unix, add separator in front
		matchingParts = append(matchingParts, string(filepath.Separator))
	}
	for i := 0; i < minLen; i++ {
		part := pathParts[0][i]
		for j := 1; j < len(pathParts); j++ {
			if pathParts[j][i] != part {
				return filepath.Join(matchingParts...), nil
			}
		}
		matchingParts = append(matchingParts, part)
	}
	// All parts matched up to the minimum length path
	return filepath.Join(matchingParts...), nil
}
