package hash_stuff

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// hash of the string "content"
var contentMd5, _ = hex.DecodeString("9a0364b9e99bb480dd25e1f0284c8555")

// hash of the string "other content"
var otherContentMd5, _ = hex.DecodeString("0c84751f0ca9c6886bb09f2dd1a66faa")

func generateTmpDir() (string, []string, error) {
	dirName, err := ioutil.TempDir("./tmp", "test-")
	if err != nil {
		return "", nil, err
	}

	rootFiles, err := generateFiles(dirName, "")
	if err != nil {
		return "", nil, err
	}

	fooDirName := filepath.Join(dirName, "foo")
	if err := os.Mkdir(fooDirName, 0777); err != nil {
		return "", nil, err
	}

	fooDirFiles, err := generateFiles(fooDirName, ".csv")
	if err != nil {
		return "", nil, err
	}

	// should have one with the extension elsewhere in the path
	tsDirName := filepath.Join(dirName, "ts-files")
	if err := os.Mkdir(tsDirName, 0777); err != nil {
		return "", nil, err
	}

	tsDirFiles, err := generateFiles(tsDirName, ".ts")
	if err != nil {
		return "", nil, err
	}

	strayTsFileName := filepath.Join(dirName, "strayTsFile.ts")
	if err := ioutil.WriteFile(strayTsFileName, []byte("content"), 0777); err != nil {
		return "", nil, err
	}

	files := append(
		rootFiles,
		append(
			fooDirFiles,
			append(tsDirFiles, strayTsFileName)...)...)

	sort.Strings(files)
	return dirName, files, nil
}

func generateFiles(dirName string, extension string) ([]string, error) {
	files := []string{}

	for i := 0; i < 10; i++ {
		strI := strconv.Itoa(i)
		path := filepath.Join(dirName, strI+extension)
		if err := ioutil.WriteFile(path, []byte("content"), 0777); err != nil {
			return nil, err
		}

		files = append(files, path)
	}

	return files, nil
}

func updateFile(t *testing.T, path string, content string) {
	if err := ioutil.WriteFile(path, []byte(content), 0777); err != nil {
		t.Fatal(err)
	}
}

func TestListFiles(t *testing.T) {
	dirName, actualFiles, err := generateTmpDir()
	defer os.RemoveAll(dirName)

	if err != nil {
		t.Fatal(err)
	}

	var expectList = func(message string, includePatterns []string, excludePatterns []string, expected []string) {
		foundFiles, err := ListFiles([]string{dirName}, includePatterns, excludePatterns)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(foundFiles, expected); diff != "" {
			t.Fatal(message, diff)
		}
	}

	expectList(
		"listFiles output should contain all files in temp dir",
		[]string{"**"},
		[]string{},
		actualFiles,
	)

	expectList(
		"listFiles output should contain just the one ts file at the root",
		[]string{"*.ts"},
		[]string{},
		[]string{filepath.Join(dirName, "strayTsFile.ts")},
	)

	expectList(
		"listFiles should exclude files",
		[]string{"*.ts"},
		[]string{"*stray*"},
		[]string{},
	)

	expectList(
		"listFiles should not list files in subdirs without a **",
		[]string{"*.csv"},
		[]string{""},
		[]string{},
	)

	expectList(
		"listFiles should list files in subdirs and exclude files",
		[]string{"**/*.csv"},
		[]string{"**/*8.csv", "**/7.*"},
		[]string{
			filepath.Join(dirName, "foo/0.csv"),
			filepath.Join(dirName, "foo/1.csv"),
			filepath.Join(dirName, "foo/2.csv"),
			filepath.Join(dirName, "foo/3.csv"),
			filepath.Join(dirName, "foo/4.csv"),
			filepath.Join(dirName, "foo/5.csv"),
			filepath.Join(dirName, "foo/6.csv"),
			filepath.Join(dirName, "foo/9.csv"),
		},
	)
}

func TestGetSummary(t *testing.T) {
	dirName, actualFiles, err := generateTmpDir()
	defer os.RemoveAll(dirName)

	if err != nil {
		t.Fatal(err)
	}

	changeFile := filepath.Join(dirName, "5")
	updateFile(t, changeFile, "other content")

	fileHashes, err := ComputeHashes(actualFiles)
	if err != nil {
		t.Fatal(err)
	}

	for _, fileHash := range fileHashes {
		expectedMd5 := contentMd5
		if fileHash.path == changeFile {
			expectedMd5 = otherContentMd5
		}

		if !cmp.Equal(fileHash.hash, expectedMd5) {
			t.Fatalf(
				`expected hash for %s to be "%s" but was "%x"`,
				fileHash.path,
				expectedMd5,
				fileHash.hash)
		}
	}

	expectedSummary := ""
	for _, fileHash := range fileHashes {
		expectedMd5 := contentMd5
		if fileHash.path == changeFile {
			expectedMd5 = otherContentMd5
		}

		expectedSummary += fmt.Sprintf("%s: %x\n", fileHash.path, expectedMd5)
	}

	if diff := cmp.Diff(GetSummary(fileHashes), expectedSummary); diff != "" {
		t.Fatal("wrong summary", diff)
	}
}

func TestGetDigest(t *testing.T) {
	dirName, _, err := generateTmpDir()
	defer os.RemoveAll(dirName)

	if err != nil {
		t.Fatal(err)
	}

	startDigest, _, err := GetDigest([]string{dirName}, []string{"**"}, []string{})
	if err != nil {
		t.Fatal(err)
	}

	changeFile := filepath.Join(dirName, "5")
	updateFile(t, changeFile, "other content")

	afterChangeDigest, _, err := GetDigest([]string{dirName}, []string{"**"}, []string{})
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(startDigest, afterChangeDigest); diff == "" {
		t.Fatalf(`expected digest to be different after changing file but both were %s`, startDigest)
	}

	updateFile(t, changeFile, "content")

	changedBackDigest, _, err := GetDigest([]string{dirName}, []string{"**"}, []string{})
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(startDigest, changedBackDigest); diff != "" {
		t.Fatal("expected digest to be the same as it was originally after changing file back", diff)
	}
}

func TestGetDigestExcludeFile(t *testing.T) {
	dirName, _, err := generateTmpDir()
	defer os.RemoveAll(dirName)

	if err != nil {
		t.Fatal(err)
	}

	startDigest, _, err := GetDigest([]string{dirName}, []string{"**"}, []string{"5"})
	if err != nil {
		t.Fatal(err)
	}

	changeFile := filepath.Join(dirName, "5")
	updateFile(t, changeFile, "other content")

	afterChangeDigest, _, err := GetDigest([]string{dirName}, []string{"**"}, []string{"5"})
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(startDigest, afterChangeDigest); diff != "" {
		t.Fatal("expected digest to not change after changing excluded file", diff)
	}
}
