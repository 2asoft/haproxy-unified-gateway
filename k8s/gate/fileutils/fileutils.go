package fileutils

import (
	"bufio"
	"os"
	"strings"
)

// ReadKeyValueFile reads a file with key-value pairs separated by a separator.
// If the file does not exist, it returns an empty map and no error.
func ReadKeyValueFile(filename string, separator rune) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	defer file.Close()

	data := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, string(separator), 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" {
				data[key] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return data, nil
}
