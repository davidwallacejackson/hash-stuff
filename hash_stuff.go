package hash_stuff

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"
)

type multiError struct {
	errorType string
	errs      []error
}

func (m multiError) Error() string {
	output := fmt.Sprintf("Errors (%s):\n", m.errorType)

	for _, err := range m.errs {
		output += fmt.Sprintf("* %s\n", err.Error())
	}

	return output
}

var _ error = multiError{}

func listFiles(rootPath string, includePatterns []string, excludePatterns []string) ([]string, error) {
	// normalize rootPath:
	rootPath = strings.TrimSuffix(rootPath, "/")

	globCompileErrors := []error{}

	includeGlobs := make([]glob.Glob, len(includePatterns))
	for i, includePattern := range includePatterns {
		compiled, err := glob.Compile(includePattern, '/')
		if err != nil {
			globCompileErrors = append(globCompileErrors, err)
		} else {
			includeGlobs[i] = compiled
		}
	}

	excludeGlobs := make([]glob.Glob, len(excludePatterns))
	for i, excludePattern := range excludePatterns {
		compiled, err := glob.Compile(excludePattern, '/')
		if err != nil {
			globCompileErrors = append(globCompileErrors, err)
		} else {
			excludeGlobs[i] = compiled
		}
	}

	if len(globCompileErrors) > 0 {
		return nil, multiError{
			errorType: "invalid glob(s)",
			errs:      globCompileErrors,
		}
	}

	matchedFilePaths := []string{}
	walkErrs := []error{}
	walkFunc := func(path string, info os.FileInfo, err error) error {
		pathWithoutPrefix := strings.TrimPrefix(path, rootPath+"/")
		if err != nil {
			// there was an error accessing this path -- log it and keep going so we can present all errors
			walkErrs = append(walkErrs, err)
			return nil
		}

		matchesInclude := matchesAny(pathWithoutPrefix, includeGlobs)
		matchesExclude := matchesAny(pathWithoutPrefix, excludeGlobs)

		if info.IsDir() {
			// recurse into all directories, unless they match an exclude
			if matchesExclude {
				return filepath.SkipDir
			}

			return nil
		}

		if matchesInclude && !matchesExclude {
			matchedFilePaths = append(matchedFilePaths, path)
		}

		// read the next file or directory
		return nil
	}

	filepath.Walk(rootPath, walkFunc)

	if len(walkErrs) > 0 {
		return nil, multiError{
			errorType: "problem listing files",
			errs:      walkErrs,
		}
	}

	return matchedFilePaths, nil
}

func matchesAny(path string, testGlobs []glob.Glob) bool {
	for _, testGlob := range testGlobs {
		if testGlob.Match(path) {
			return true
		}
	}

	return false
}
