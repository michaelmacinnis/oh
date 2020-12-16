package history

import (
	"io"
	"os"
)

// Load loads any saved command history.
func Load(read func(r io.Reader) (int, error)) error {
	f, err := file(os.Open)
	if err != nil {
		// We may not find a history file.
		return nil
	}

	_, err = read(f)
	if err != nil {
		return err
	}

	return f.Close()
}

// Save saves the current command history.
func Save(write func(w io.Writer) (int, error)) error {
	f, err := file(os.Create)
	if err != nil {
		return err
	}

	_, err = write(f)
	if err != nil {
		return err
	}

	return f.Close()
}
