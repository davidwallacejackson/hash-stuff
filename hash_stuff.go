package hash_stuff

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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
	globCompileErrors := []error{}

	log.Println("Parsing include patterns")
	includeGlobs := make([]glob.Glob, len(includePatterns))
	for i, includePattern := range includePatterns {
		compiled, err := glob.Compile(includePattern)
		if err != nil {
			globCompileErrors = append(globCompileErrors, err)
		} else {
			includeGlobs[i] = compiled
		}
	}

	log.Println("Parsing exclude patterns")
	excludeGlobs := make([]glob.Glob, len(excludePatterns))
	for i, excludePattern := range excludePatterns {
		compiled, err := glob.Compile(excludePattern)
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

	log.Printf("Walking root path %s", rootPath)
	matchedFilePaths := []string{}
	walkErrs := []error{}
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// there was an error accessing this path -- log it and keep going so we can present all errors
			walkErrs = append(walkErrs, err)
			return nil
		}

		matchesInclude := matchesAny(path, includeGlobs)
		matchesExclude := matchesAny(path, excludeGlobs)

		if info.IsDir() {
			if matchesInclude && !matchesExclude {
				// this is a directory we should be checking -- desceend into it and keep going
				return nil
			}

			return filepath.SkipDir
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
