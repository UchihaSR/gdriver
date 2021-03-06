package driveext

import (
	"crypto/md5"
	"fmt"
	"github.com/dustin/go-humanize"
	"io"
	"os"

	"github.com/pkg/errors"
)

type FileState struct {
	File      *DriveFile
	LocalPath string

	missing     bool
	md5Checksum string
	size        int64
}

func (fs *FileState) Offset() int64 {
	if fs.size > fs.File.Size {
		return 0 // file is corrupted (too long)
	}

	if fs.size == fs.File.Size && fs.md5Checksum != fs.File.Md5Checksum {
		return 0 // file is corrupted
	}
	return fs.size
}

func (fs *FileState) Valid() (bool, error) {
	if fs.missing {
		return false, errors.New("missing file")
	}

	if fs.size == 0 {
		return false, errors.New("empty file")
	}

	if fs.size < fs.File.Size {
		return false, fmt.Errorf("incomplete file (%s / %s)",
			humanize.Bytes(uint64(fs.size)), humanize.Bytes(uint64(fs.File.Size)))
	}

	if fs.md5Checksum != fs.File.Md5Checksum {
		return false, fmt.Errorf("corrupted file (bad MD5 checksum: %s)", fs.md5Checksum)
	}
	return true, nil
}

func EvaluateFileState(file *DriveFile, localPath string) (*FileState, error) {
	state := &FileState{
		File:      file,
		LocalPath: localPath,
	}

	fi, err := os.Stat(localPath)
	if os.IsNotExist(err) {
		state.missing = true
		return state, nil // file hasn't been downloaded yet
	}
	if err != nil {
		return nil, errors.Wrapf(err, "stat file failed (path: %s)", state.LocalPath)
	}

	state.md5Checksum, err = calculateMd5Checksum(localPath)
	if err != nil {
		return nil, errors.Wrap(err, "calculating MD5 checksum failed")
	}

	state.size = fi.Size()
	return state, nil
}

func calculateMd5Checksum(localPath string) (string, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return "", errors.Wrapf(err, "can't open file (path: %s)", localPath)
	}

	h := md5.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", errors.Wrap(err, "copying buffer failed")
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
