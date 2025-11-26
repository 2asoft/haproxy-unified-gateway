package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/stretchr/testify/assert/yaml"
)

// content of the file
// hug-conf.go:
//   v0.7.0: ""
// gate.go:
//   v0.7.0: ""

type VersionsForHugConf struct {
	Versions map[string]string `json:"versions"`
}

func main() {
	// this program will open all go source files in api/gate/v3/
	// if file contains // haproxy.org/version=" string, it will print the filename and the version found
	files, err := filepath.Glob("../../api/gate/v3/*.go")
	if err != nil {
		log.Fatal(err)
	}
	foundMismatch := false
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}

		if strings.Contains(string(data), "hug/version=") {
			// extract the version string
			start := strings.Index(string(data), "hug/version=")
			end := strings.Index(string(data)[start+len("hug/version="):], `"`)
			version := string(data)[start+len("hug/version=") : start+len("hug/version=")+end]
			// calculate hash of the file content
			hash := sha256.Sum256(data)

			// fmt.Printf("%s %s %s\n", filepath.Base(file), version, fmt.Sprintf("%x", hash))

			// now check if the version exists in api/gate/v3/versions.yml
			versionsData, err := os.ReadFile("../../api/gate/v3/versions.yml")
			if err != nil {
				log.Fatal(err)
			}
			// convert that to struct
			var versions map[string]map[string]string
			err = yaml.Unmarshal(versionsData, &versions)
			if err != nil {
				log.Fatal(err)
			}
			fileName := filepath.Base(file)
			if v, ok := versions[fileName]; ok {
				fileHash := v[version]
				if fileHash != fmt.Sprintf("%x", hash) {
					foundMismatch = true
					fmt.Printf("Version hash mismatch for %s version %s: expected %s, got %s\n", fileName, version, fileHash, fmt.Sprintf("%x", hash))
				}
			}
		}
	}
	if foundMismatch {
		os.Exit(1)
	}
}
