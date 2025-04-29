package mp4

import (
	"bytes"
	"io"
	"os"

	"github.com/alfg/mp4/atom"
)

// Open opens a file and returns an &Mp4Reader{}.
func Open(path string) (f *atom.Mp4Reader, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	f = &atom.Mp4Reader{
		Reader: file,
	}
	return f, f.Parse()
}

// OpenFromReader read and returns an &Mp4Reader{}.
func OpenFromReader(reader io.ReaderAt, size int64) (f *atom.Mp4Reader, err error) {
	f = &atom.Mp4Reader{
		Reader: reader,
		Size:   int64(size),
	}
	return f, f.Parse()
}

// OpenFromBytes read and returns an &Mp4Reader{}.
func OpenFromBytes(buffer []byte) (f *atom.Mp4Reader, err error) {
	f = &atom.Mp4Reader{
		Reader: bytes.NewReader(buffer),
		Size:   int64(len(buffer)),
	}
	return f, f.Parse()
}
