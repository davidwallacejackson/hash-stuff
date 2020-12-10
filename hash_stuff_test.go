package hash_stuff

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func generateTmpDir() (string, []string, error) {
	dirName, err := ioutil.TempDir("./tmp", "test-")
	if err != nil {
		return "", nil, err
	}

	files, err := generateFiles(dirName)
	if err != nil {
		return "", nil, err
	}

	return dirName, files, nil
}

func generateFiles(dirName string) ([]string, error) {
	files := []string{}

	for i := 0; i < 1000; i++ {
		strI := strconv.Itoa(i)
		path := filepath.Join(dirName, strI)
		if err := ioutil.WriteFile(path, []byte("content"), 0777); err != nil {
			return nil, err
		}

		files = append(files, path)
	}

	sort.Strings(files)
	return files, nil
}

func TestListFiles(t *testing.T) {
	dirName, actualFiles, err := generateTmpDir()
	defer os.RemoveAll(dirName)

	if err != nil {
		t.Fatal(err)
	}

	foundFiles, err := listFiles(dirName, []string{"**"}, []string{})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(foundFiles, actualFiles); diff != "" {
		t.Fatal("listFiles output should contain all files in temp dir", diff)
	}
}
