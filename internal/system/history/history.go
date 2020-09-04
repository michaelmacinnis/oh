package history

import (
	"io"
	"os"
)

func Load(read func(r io.Reader) (int, error)) error {
	f, err := file(os.Open)
	if err != nil {
		return err
	}

	_, err = read(f)
	if err != nil {
		return err
	}

	return f.Close()
}

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
