// Copyright 2025 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package fileutils

import (
	"bufio"
	"os"
	"path/filepath"
)

// FilePath represents a file in a specific directory.
type FilePath struct {
	Dir      string
	FileName string
}

// FullPath returns the complete path by joining BaseDir and FileName.
func (fp FilePath) FullPath() string {
	return filepath.Join(fp.Dir, fp.FileName)
}

func (fp FilePath) DeleteFromDisk() error {
	fullPath := fp.FullPath()
	return os.Remove(fullPath)
}

// ReadLines reads a file and returns its content as a slice of strings,
// with each string representing a line from the file.
func (fp FilePath) ReadLines() ([]string, error) {
	// Open the file.
	file, err := os.Open(fp.FullPath())
	if err != nil {
		return nil, err
	}
	// Ensure the file is closed when the function returns.
	defer file.Close()

	// Create a slice to hold the lines.
	var lines []string
	// Use a bufio.Scanner to efficiently read the file line by line.
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Check for any errors that occurred during scanning.
	return lines, scanner.Err()
}
