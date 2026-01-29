package parser

import (
	"os"
)

// openFile opens a file for reading.
func openFile(path string) (*os.File, error) {
	return os.Open(path)
}
