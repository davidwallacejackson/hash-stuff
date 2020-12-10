package hash_stuff

import (
	"io/ioutil"
	"path/filepath"
	"strconv"
)

func generateFiles() (string, error) {
	dirName, err := ioutil.TempDir("./tmp", "test-")
	if err != nil {
		return "", err
	}

	for i := 0; i < 1000; i++ {
		strI := strconv.Itoa(i)
		if err := ioutil.WriteFile(filepath.Join(dirName, strI), []byte("test "+strI), 0777); err != nil {
			return "", err
		}
	}

	return dirName, nil
}
