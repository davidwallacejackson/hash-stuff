package hash_stuff

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

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

func ListFiles(rootPaths []string, includePatterns []string, excludePatterns []string) ([]string, error) {
	var matchedFilePaths []string = []string{}
	var rootPathErrors []error

	for _, rootPath := range rootPaths {
		matchedFilePathsForRootPath, err := listFilesInternal(rootPath, includePatterns, excludePatterns)
		if err != nil {
			rootPathErrors = append(rootPathErrors, err)
		}

		matchedFilePaths = append(matchedFilePaths, matchedFilePathsForRootPath...)
	}

	if len(rootPathErrors) > 0 {
		return nil, multiError{
			errorType: "error in one or more root paths",
			errs:      rootPathErrors,
		}
	}

	sort.Strings(matchedFilePaths)

	return matchedFilePaths, nil
}

func listFilesInternal(rootPath string, includePatterns []string, excludePatterns []string) ([]string, error) {
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

type fileHash struct {
	path string
	hash []byte
}

func ComputeHashes(paths []string) ([]fileHash, error) {
	var wg sync.WaitGroup

	fileHashes := make([]fileHash, len(paths))
	var errs []error

	for i, path := range paths {
		wg.Add(1)
		go func(i int, path string) {
			hash, err := hashFile(path)
			if err != nil {
				// TODO: check if this is actually safe?
				errs = append(errs, err)
			} else {
				fileHashes[i] = fileHash{
					path: path,
					hash: hash,
				}
			}

			wg.Done()
		}(i, path)
	}

	wg.Wait()

	if len(errs) > 0 {
		return nil, multiError{
			errorType: "problem computing hashes",
			errs:      errs,
		}
	}

	return fileHashes, nil
}

func hashFile(path string) ([]byte, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return hashString(string(contents))
}

func hashString(hashMe string) ([]byte, error) {
	h := md5.New()
	_, err := h.Write([]byte(hashMe))
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

func GetSummary(fileHashes []fileHash) string {
	summary := ""

	for _, fileHash := range fileHashes {
		summary += fmt.Sprintf("%s: %x\n", fileHash.path, fileHash.hash)
	}

	return summary
}

func GetDigest(rootPaths []string, includePatterns []string, excludePatterns []string) ([]byte, string, error) {
	paths, err := ListFiles(rootPaths, includePatterns, excludePatterns)
	if err != nil {
		return nil, "", err
	}

	fileHashes, err := ComputeHashes(paths)
	if err != nil {
		return nil, "", err
	}

	summary := GetSummary(fileHashes)

	digest, err := hashString(summary)
	if err != nil {
		return nil, "", err
	}

	return digest, summary, nil
}
