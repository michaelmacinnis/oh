// Released under an MIT-style license. See LICENSE.

package task

import (
	"os"
	"strings"
)

func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return os.ErrPermission
}

func lookPath(file string) (string, error) {
	cnf := "command not found"

	// Only bypass the path if file begins with / or ./ or ../
	prefix := file + "   "
	if prefix[0:1] == "/" || prefix[0:2] == "./" || prefix[0:3] == "../" {
		err := findExecutable(file)
		if err == nil {
			return file, nil
		}
		return "", &pathError{file, err.Error()}
	}
	pathenv := os.Getenv("PATH")
	if pathenv == "" {
		return "", &pathError{file, cnf}
	}
	for _, dir := range strings.Split(pathenv, ":") {
		path := dir + "/" + file
		if err := findExecutable(path); err == nil {
			return path, nil
		}
	}
	return "", &pathError{file, cnf}
}
