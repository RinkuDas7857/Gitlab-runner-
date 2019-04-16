package archives

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testZipFileContent = []byte("test content")

type CharsetByte int

const (
	SingleByte CharsetByte = iota
	MultiBytes
)

func createTestFile(t *testing.T, csb CharsetByte) string {
	name := "test_file.txt"
	if csb == MultiBytes {
		name = "テストファイル.txt"
	}

	err := ioutil.WriteFile(name, testZipFileContent, 0640)
	assert.NoError(t, err)
	return name
}

func createSymlinkFile(t *testing.T, csb CharsetByte) string {
	name := "new_symlink"
	if csb == MultiBytes {
		name = "新しいシンボリックリンク"
	}

	err := os.Symlink("old_symlink", name)
	assert.NoError(t, err)
	return name
}

func createTestDirectory(t *testing.T, csb CharsetByte) string {
	name := "test_directory"
	if csb == MultiBytes {
		name = "テストディレクトリ"
	}

	err := os.Mkdir(name, 0711)
	assert.NoError(t, err)
	return name
}

func createTestPipe(t *testing.T, csb CharsetByte) string {
	name := "test_pipe"
	if csb == MultiBytes {
		name = "テストパイプ"
	}

	err := syscall.Mkfifo(name, 0600)
	assert.NoError(t, err)
	return name
}

func createTestGitPathFile(t *testing.T, csb CharsetByte) string {
	_, err := os.Stat(".git")
	if err != nil {
		err = os.Mkdir(".git", 0711)
		assert.NoError(t, err)
	}

	name := ".git/test_file"
	if csb == MultiBytes {
		name = ".git/テストファイル"
	}

	err = ioutil.WriteFile(name, testZipFileContent, 0640)
	assert.NoError(t, err)

	return name
}

func testInWorkDir(t *testing.T, testCase func(t *testing.T, fileName string)) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(wd) }()

	td, err := ioutil.TempDir("", "zip_create")
	require.NoError(t, err)

	err = os.Chdir(td)
	assert.NoError(t, err)

	tempFile, err := ioutil.TempFile("", "archive")
	require.NoError(t, err)
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	testCase(t, tempFile.Name())
}

func TestZipCreate(t *testing.T) {
	testInWorkDir(t, func(t *testing.T, fileName string) {
		paths := []string{
			createTestFile(t, SingleByte),
			createSymlinkFile(t, SingleByte),
			createTestDirectory(t, SingleByte),
			createTestPipe(t, SingleByte),
			createTestFile(t, MultiBytes),
			createSymlinkFile(t, MultiBytes),
			createTestDirectory(t, MultiBytes),
			createTestPipe(t, MultiBytes),
			"non_existing_file.txt",
		}
		err := CreateZipFile(fileName, paths)
		require.NoError(t, err)

		archive, err := zip.OpenReader(fileName)
		require.NoError(t, err)
		defer archive.Close()

		assert.Len(t, archive.File, 6)

		assert.Equal(t, "test_file.txt", archive.File[0].Name)
		assert.Equal(t, os.FileMode(0640), archive.File[0].Mode().Perm())
		assert.NotEmpty(t, archive.File[0].Extra)

		assert.Equal(t, "new_symlink", archive.File[1].Name)

		assert.Equal(t, "test_directory/", archive.File[2].Name)
		assert.NotEmpty(t, archive.File[2].Extra)
		assert.True(t, archive.File[2].Mode().IsDir())

		assert.Equal(t, "テストファイル.txt", archive.File[3].Name)
		assert.Equal(t, os.FileMode(0640), archive.File[3].Mode().Perm())
		assert.NotEmpty(t, archive.File[3].Extra)

		assert.Equal(t, "新しいシンボリックリンク", archive.File[4].Name)

		assert.Equal(t, "テストディレクトリ/", archive.File[5].Name)
		assert.NotEmpty(t, archive.File[5].Extra)
		assert.True(t, archive.File[5].Mode().IsDir())
	})
}

func TestZipCreateWithGitPath(t *testing.T) {
	testInWorkDir(t, func(t *testing.T, fileName string) {
		output := logrus.StandardLogger().Out
		var buf bytes.Buffer
		logrus.SetOutput(&buf)
		defer logrus.SetOutput(output)

		paths := []string{
			createTestGitPathFile(t, SingleByte),
			createTestGitPathFile(t, MultiBytes),
		}
		err := CreateZipFile(fileName, paths)
		require.NoError(t, err)

		assert.Contains(t, buf.String(), "Part of .git directory is on the list of files to archive")

		archive, err := zip.OpenReader(fileName)
		require.NoError(t, err)
		defer archive.Close()

		assert.Len(t, archive.File, 2)

		assert.Equal(t, ".git/test_file", archive.File[0].Name)
		assert.Equal(t, os.FileMode(0640), archive.File[0].Mode().Perm())
		assert.NotEmpty(t, archive.File[0].Extra)

		assert.Equal(t, ".git/テストファイル", archive.File[1].Name)
		assert.Equal(t, os.FileMode(0640), archive.File[1].Mode().Perm())
		assert.NotEmpty(t, archive.File[1].Extra)
	})
}
