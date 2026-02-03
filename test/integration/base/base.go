// Copyright 2019 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package base

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	futils "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/fileutils"
	"sigs.k8s.io/yaml"

	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/haproxy-unified-gateway/test/integration/utils"
	"github.com/stretchr/testify/suite"
)

const (
	timeout  = time.Second * 15
	interval = time.Second * 1
)

type BaseSuite struct {
	suite.Suite
	test IntTest
}

func (b *BaseSuite) Test() IntTest {
	return b.test
}

func (b *BaseSuite) SetupSuite() {
	var err error
	b.test, err = NewIntTest(b.T())
	b.Require().NoError(err)

	b.test.StartTestEnv(b.T())
}

func (b *BaseSuite) TearDownSuite() {
	b.test.StopTestEnv(b.T())
}

// CreateFixtures will create all the objects from manifests that are in the fixturePath directory
// They are created in the test namespace
// To create in a specific namespace use CreateFixturesInNamespace
// If manifestNames is empty, it will create all objects that are in the directory
// If manifestNames, it will create only the objects in manifestNames (a sublist of the files in fixturePath)
func (b *BaseSuite) CreateFixtures(fixturePath string, manifestNames []string) {
	params := utils.RuntimeYamlParams{
		Ctx:               b.Test().Ctx,
		CrtlruntimeClient: b.Test().Client,
		Namespace:         b.Test().Namespace,
		Dir:               fixturePath,
		WaitForResult:     true,
		ManifestNames:     manifestNames,
	}
	err := utils.CreateRuntimeObjectsFromYAMLFiles(params)
	b.Require().NoError(err)
}

func (b *BaseSuite) CreateFixturesInNamespace(fixturePath, namespace string, manifestNames []string) {
	params := utils.RuntimeYamlParams{
		Ctx:               b.Test().Ctx,
		CrtlruntimeClient: b.Test().Client,
		Namespace:         namespace,
		Dir:               fixturePath,
		WaitForResult:     true,
		ManifestNames:     manifestNames,
	}
	err := utils.CreateRuntimeObjectsFromYAMLFiles(params)
	b.Require().NoError(err)
}

func (b *BaseSuite) CleanupFixtures(fixturePath string, manifestNames []string) {
	params := utils.RuntimeYamlParams{
		Ctx:               b.Test().Ctx,
		CrtlruntimeClient: b.Test().Client,
		Namespace:         b.Test().Namespace,
		Dir:               fixturePath,
		WaitForResult:     true,
		ManifestNames:     manifestNames,
	}
	err := utils.DeleteRuntimeObjectsFromYAMLFiles(params)
	b.Require().NoError(err)
}

func (b *BaseSuite) CleanupFixturesInNamespace(fixturePath, namespace string, manifestNames []string) {
	params := utils.RuntimeYamlParams{
		Ctx:               b.Test().Ctx,
		CrtlruntimeClient: b.Test().Client,
		Namespace:         namespace,
		Dir:               fixturePath,
		WaitForResult:     true,
		ManifestNames:     manifestNames,
	}
	err := utils.DeleteRuntimeObjectsFromYAMLFiles(params)
	b.Require().NoError(err)
}

// Return HAProxy master process if it exists.
// func haproxyProcess(pidFile string) (*os.Process, error) {
// 	file, err := os.Open(pidFile)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()
// 	scanner := bufio.NewScanner(file)
// 	scanner.Scan()
// 	pid, err := strconv.Atoi(scanner.Text())
// 	if err != nil {
// 		return nil, err
// 	}
// 	process, err := os.FindProcess(pid)
// 	if err != nil {
// 		return nil, err
// 	}
// 	err = process.Signal(syscall.Signal(0))
// 	return process, err
// }

// For now: only checks the following certificate fields:
// - StorageName
// - Subject (for CN)
// Check is performed using the RUNTIME command on haproxy
func (b *BaseSuite) ExpectCertificates(ctx context.Context, expectedCerts []*models.SslCertificate) {
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		// Runtime command to get the list of certs
		// Warning: the returned certs are not filled completely, only the StorageName and Description
		certs, err := b.Test().RuntimeClient.ShowCerts()
		if err != nil {
			return false
		}

		if len(expectedCerts) != len(certs) {
			fmt.Printf("len(expectedCerts): %v\n", len(expectedCerts))
			fmt.Printf("len(certs): %v\n", len(certs))
			return false
		}
		gotCertSkeletonMap := make(map[string]*models.SslCertificate)
		for _, cert := range certs {
			gotCertSkeletonMap[cert.StorageName] = cert
		}

		for _, expectedCert := range expectedCerts {
			if _, ok := gotCertSkeletonMap[expectedCert.StorageName]; !ok {
				return false
			}
			// Runtime command to actually get the Cert content
			gotCert, err := b.Test().RuntimeClient.ShowCertificate(expectedCert.StorageName)
			b.Require().NoError(err)
			areOK := b.areCertsEqual(expectedCert, gotCert)
			if !areOK {
				return false
			}
		}
		return true
	}) {
		b.T().Fatal("certificates not correct")
	}
}

// For now: only checks the following certificate fields:
// - StorageName
// - Subject (for CN)
func (*BaseSuite) areCertsEqual(expected, got *models.SslCertificate) bool {
	// Do they have the same CN ?
	return expected.StorageName == got.StorageName && expected.Subject == got.Subject
}

// ExpectCrtLists checks that the crt-list are the expectedCrtLists
// For now, there is no CN method to get the content of the crt-list : "show ssl crt-list <filename>"
// The only existing method in CN is "show ssl crt-list" that gives the list of crt-lists
// So, for now, we check the content of the crt-list from the crt-list file, not from the RUNTIME.
func (b *BaseSuite) ExpectCrtLists(ctx context.Context, expectedCrtLists map[futils.FilePath][]string) {
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		// Runtime command to get the list of crtLists
		crtLists, err := b.Test().RuntimeClient.ShowCrtLists()
		if err != nil {
			return false
		}

		if len(expectedCrtLists) != len(crtLists) {
			return false
		}
		gotCrtListSkeletonMap := make(map[string]*models.SslCrtList)
		for _, crtList := range crtLists {
			gotCrtListSkeletonMap[crtList.File] = crtList
		}

		for crtListFilePath, expectedCrtListContent := range expectedCrtLists {
			if _, ok := gotCrtListSkeletonMap[crtListFilePath.FullPath()]; !ok {
				return false
			}
			gotContent, err := crtListFilePath.ReadLines()
			b.Require().NoError(err)
			sort.Strings(gotContent)
			// Runtime command to actually get the CrtList content
			areEqual := b.areCrtListContentEqual(expectedCrtListContent, gotContent)
			if !areEqual {
				return false
			}
		}
		return true
	}) {
		b.T().Fatal("crt-list not correct")
	}
}

func (*BaseSuite) areCrtListContentEqual(expected, got []string) bool {
	if len(expected) != len(got) {
		return false
	}
	for i := range expected {
		if expected[i] != got[i] {
			return false
		}
	}
	return true
}

func (b *BaseSuite) ExpectFrontends(ctx context.Context, expectationPath string, expectedFrontends []string) {
	var diffs map[string][]any
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		frontends, err := b.test.HaproxyClient.FrontendsGet()
		if err != nil {
			return false
		}

		gotFrontends := make(map[string]*models.Frontend)
		for _, fe := range frontends {
			gotFrontends[fe.Name] = fe
			// b.exportFrontend(fe)
		}

		for _, expectedFeName := range expectedFrontends {
			var gotFrontend *models.Frontend
			var ok bool
			if gotFrontend, ok = gotFrontends[expectedFeName]; !ok {
				return false
			}

			expectedFrontend := b.FrontendFromManifest(expectationPath, expectedFeName)
			areSame := expectedFrontend.Equal(*gotFrontend)
			if !areSame {
				diffs = expectedFrontend.Diff(*gotFrontend)
				return false
			}
		}
		return true
	}) {
		b.T().Fatalf("frontends diffs\n %v ", diffs)
	}
}

func (b *BaseSuite) FrontendFromManifest(manifestPath, manifestName string) *models.Frontend {
	mpath := path.Join(manifestPath, manifestName+".yaml")
	yamlFile, err := os.ReadFile(mpath)
	b.Require().NoError(err)

	var fe models.Frontend
	err = yaml.Unmarshal(yamlFile, &fe)
	b.Require().NoError(err)
	return &fe
}

func (b *BaseSuite) ExpectBackends(ctx context.Context, expectationPath string, expectedBackends []string) {
	var diffs map[string][]any
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		backends, err := b.test.HaproxyClient.BackendsGet()
		if err != nil {
			return false
		}

		gotBackends := make(map[string]*models.Backend)
		for _, be := range backends {
			gotBackends[be.Name] = be
			// b.exportBackend(be)
		}

		for _, expectedBeName := range expectedBackends {
			var gotBackend *models.Backend
			var ok bool
			if gotBackend, ok = gotBackends[expectedBeName]; !ok {
				return false
			}

			expectedBackend := b.BackendFromManifest(expectationPath, expectedBeName)
			areSame := expectedBackend.Equal(*gotBackend)
			if !areSame {
				diffs = expectedBackend.Diff(*gotBackend)
				return false
			}
		}
		return true
	}) {
		b.T().Fatalf("backends diffs\n %v ", diffs)
	}
}

func (b *BaseSuite) ExpectBackendsDoNotExist(ctx context.Context, backendThatShouldNotExist string) {
	var diffs map[string][]any
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		backends, err := b.test.HaproxyClient.BackendsGet()
		if err != nil {
			return false
		}

		gotBackends := make(map[string]*models.Backend)
		for _, be := range backends {
			gotBackends[be.Name] = be
		}

		if _, ok := gotBackends[backendThatShouldNotExist]; !ok {
			return true
		}

		return false
	}) {
		b.T().Fatalf("backends diffs\n %v ", diffs)
	}
}

func (b *BaseSuite) BackendFromManifest(manifestPath, manifestName string) *models.Backend {
	mpath := path.Join(manifestPath, manifestName+".yaml")
	yamlFile, err := os.ReadFile(mpath)
	b.Require().NoError(err)

	var be models.Backend
	err = yaml.Unmarshal(yamlFile, &be)
	b.Require().NoError(err)
	return &be
}

func (b *BaseSuite) GetMapFileFrom(mapFileRelativePath string) ([]string, error) {
	var mapFile []byte
	mapFile, err := os.ReadFile(filepath.Join(b.test.HaproxyCfgDir, "maps", mapFileRelativePath))
	if err != nil {
		return nil, err
	}
	return strings.Split(string(mapFile), "\n"), nil
}

func (b *BaseSuite) CheckEntryInMapFile(mapFileRelativePath, key, value string) bool {
	mapFile, err := b.GetMapFileFrom(mapFileRelativePath)
	if err != nil {
		return false
	}
	for _, line := range mapFile {
		if line == key+" "+value {
			return true
		}
	}
	return false
}

var StandardMaps = []string{
	"domain_wildcard_path_exact.map",
	"domain_wildcard_sni.map",
	"path_exact.map",
	"path_prefix.map",
	"path_regex.map",
	"sni.map",
}

func (b *BaseSuite) CheckMapContents(mapFileRelativePath, expectedMapPath string) bool {
	for _, mapName := range StandardMaps {
		expectedFilePath := path.Join(expectedMapPath, mapName)

		// Check if expectation exists
		expectedContent, err := os.ReadFile(expectedFilePath)
		expectationExists := err == nil

		// Read actual map
		actualMapPath := filepath.Join(b.test.HaproxyCfgDir, "maps", mapFileRelativePath, mapName)
		actualContent, err := os.ReadFile(actualMapPath)
		// If actual map doesn't exist, we treat it as empty string
		var actualString string
		if err == nil {
			actualString = string(actualContent)
		}

		if expectationExists {
			// Check if content matches
			if string(expectedContent) != actualString {
				b.T().Logf("Map mismatch for %s: expected %q, got %q", mapName, string(expectedContent), actualString)
				return false
			}
		} else {
			// Check if actual is empty
			if strings.TrimSpace(actualString) != "" {
				b.T().Logf("Map %s should be empty but has content: %q", mapName, actualString)
				return false
			}
		}
	}
	return true
}

// func (b *BaseSuite) exportFrontend(fe *models.Frontend) {
// 	// Marshal the Go struct into a YAML byte slice.
// 	// This process converts the Go data structure into its YAML representation.
// 	jsonData, err := json.Marshal(fe)
// 	if err != nil {
// 		log.Fatalf("Error marshaling to YAML: %v", err)
// 	}
// 	// Define the output file name.
// 	filePath := fe.Name + ".yaml"

// 	// Write the YAML data to the file.
// 	// os.WriteFile is a convenience function that creates the file if it doesn't exist,
// 	// writes the data, and closes the file.
// 	err = os.WriteFile(filePath, jsonData, 0o644)
// 	if err != nil {
// 		log.Fatalf("Error writing file: %v", err)
// 	}
// }

// func (b *BaseSuite) exportBackend(be *models.Backend) {
// 	// Marshal the Go struct into a YAML byte slice.
// 	// This process converts the Go data structure into its YAML representation.
// 	jsonData, err := json.Marshal(be)
// 	if err != nil {
// 		log.Fatalf("Error marshaling to YAML: %v", err)
// 	}
// 	// Define the output file name.
// 	filePath := be.Name + ".yaml"

// 	// Write the YAML data to the file.
// 	// os.WriteFile is a convenience function that creates the file if it doesn't exist,
// 	// writes the data, and closes the file.
// 	err = os.WriteFile(filePath, jsonData, 0o644)
// 	if err != nil {
// 		log.Fatalf("Error writing file: %v", err)
// 	}
// }
