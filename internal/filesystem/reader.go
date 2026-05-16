package filesystem

import "os"

// Reader reads file content by path.
type Reader interface {
	ReadFile(path string) ([]byte, error)
}

// OSReader reads files from the local filesystem.
type OSReader struct{}

// ReadFile reads file content from the local filesystem.
func (OSReader) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// ReadOptionalFile reads a file and treats a missing path as empty content.
func ReadOptionalFile(reader Reader, path string) ([]byte, error) {
	if reader == nil {
		reader = OSReader{}
	}
	data, err := reader.ReadFile(path)
	if err == nil {
		return data, nil
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return nil, err
}
