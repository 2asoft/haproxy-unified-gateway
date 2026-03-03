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
	"slices"
	"strings"
	"time"

	rc "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/conditions/routes"
	futils "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/fileutils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/haproxytech/client-native/v6/models"
	truntime "github.com/haproxytech/haproxy-unified-gateway/test/integration/runtime"
	"github.com/haproxytech/haproxy-unified-gateway/test/integration/utils"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

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

func (b *BaseSuite) SetupSuite(crdRelativePath string, levelsUp int) {
	var err error
	b.test, err = NewIntTest(b.T(), crdRelativePath, levelsUp)
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
			slices.Sort(gotContent)
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
	return slices.Contains(mapFile, key+" "+value)
}

var StandardMaps = []string{
	"domain_wildcard_path_exact.map",
	"domain_wildcard_sni.map",
	"path_exact.map",
	"path_prefix.map",
	"path_regex.map",
	"sni.map",
}

func (b *BaseSuite) ExpectMapContents(mapFilePath, expectedMapPath string) {
	res := b.Eventually(func() bool {
		check := b.CheckMapContents(mapFilePath, expectedMapPath)
		return check
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))
	if !res {
		msg := fmt.Sprintf("maps in %s did not match expected contents", mapFilePath)
		b.T().Fatal(msg)
	}
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
				b.T().Logf("Map mismatch for %s: \nexpected %q, \ngot      %q", mapName, string(expectedContent), actualString)
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

func (b *BaseSuite) ExpectRouteConditionsUpdated(ctx context.Context, namespace, name string, expectedConditions rc.RouteConditions) {
	route := &gatewayv1.HTTPRoute{}
	var gotConditions rc.RouteConditions
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		if err := b.Test().Client.Get(
			b.Test().Ctx,
			types.NamespacedName{Name: name, Namespace: namespace}, route); err != nil {
			return false
		}

		gotConditions = rc.NewRouteConditionsFromV1RouteConditions(route.Status.Parents, TestControllerName)

		res := gotConditions.Equal(expectedConditions)

		return res
	}) {
		b.T().Fatalf("conditions not correct,\nGot %+v\nExpected %+v\n", gotConditions, expectedConditions)
	}
}

func (b *BaseSuite) ExpectAttachedRoute(ctx context.Context, namespace, gwName, listenerName string, expectNbAttachedRoutes int32) {
	gw := &gatewayv1.Gateway{}
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		if err := b.Test().Client.Get(
			b.Test().Ctx,
			types.NamespacedName{Name: gwName, Namespace: namespace}, gw); err != nil {
			return false
		}

		for _, listenerStatus := range gw.Status.Listeners {
			if string(listenerStatus.Name) == listenerName {
				return listenerStatus.AttachedRoutes == expectNbAttachedRoutes
			}
		}

		return false
	}) {
		b.T().Fatal("AttachedRoutes not correct")
	}
}

func (b *BaseSuite) ExpectServers(backend string, expectedServers []string) {
	var servers []string
	res := b.Eventually(func() bool {
		var err error
		servers, err = truntime.GetServers(b.test.RuntimeSocketPath, backend)
		if err != nil {
			return false
		}
		slices.Sort(servers)
		slices.Sort(expectedServers)
		return slices.Equal(servers, expectedServers)
	}, timeout, interval, fmt.Sprintf("servers in backend %s did not match expected", backend))
	if !res {
		msg := fmt.Sprintf("servers in backend %s did not match expected. Got %v, expected %v", backend, servers, expectedServers)
		b.T().Fatal(msg)
	}
}

// WaitForNoReloadsAnyMore checks during an overall overAllDuration
// That we reach a stable state without reloads for at least consistentlyDurationWithoutReloads
// We set a timer stabilityTimer and if at any point during the stability window a reload occurs, we reset this timer.
// If we reach the overAllDuration without a stable window without reload we issue a t.Fatalf()
func (b *BaseSuite) WaitForNoReloadsAnyMore(
	overAllDuration time.Duration,
	consistentlyDurationWithoutReloads time.Duration,
) string {
	fmt.Printf("\n...wait for a stability window without reloads: %v\n", consistentlyDurationWithoutReloads)

	info, err := truntime.GetGlobalHAProxyInfo(b.test.RuntimeSocketPath)
	pid := info.Pid
	if err != nil {
		b.T().Fatalf("error getting HAProxy info: %v", err)
	}
	overallTimeout := time.After(overAllDuration)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// stabilityTimer is reset each time there is a reload
	// It's a sliding window
	// We want to have a stable state without reloads for a least consistentlyDurationWithoutReloads
	var stabilityTimer *time.Timer
	inStabilityWindow := false

	shouldContinue := true
	for shouldContinue {
		select {
		case <-overallTimeout:
			b.T().Fatalf("Timed out after %v without reaching stable state without reloads", overAllDuration)
		case <-ticker.C:
			reloaded, newPid, err := b.haproxyReloadHappened(pid)
			if err != nil {
				b.T().Log(err)
				continue
			}
			pid = newPid
			if reloaded {
				// reload happened; if we were counting stability, reset.
				if inStabilityWindow {
					stabilityTimer.Stop()
					inStabilityWindow = false
					fmt.Print("reload happened; stopping stability timer...\n")
				}
				continue
			}
			// no reload
			if !inStabilityWindow {
				// start the stability countdown
				inStabilityWindow = true
				stabilityTimer = time.NewTimer(consistentlyDurationWithoutReloads)
				fmt.Print("no reload; starting stability timer...\n")
				continue
			}
			// already in stability window, check if time is up
			select {
			case <-stabilityTimer.C:
				// SUCCESS: stable for the full duration
				shouldContinue = false
				fmt.Printf("no reload in %v... stable state reached...\n", consistentlyDurationWithoutReloads)
			default:
				// still waiting for stability
			}
		}
	}
	return pid
	// Exit the loop means that we were stable without reload for consistentlyDurationWithoutReloads duration
}

// haproxyReloadHappened returns:
// - a bool true if a reload did happen
// - the new pid
// - an error if we could get the worker pid
// The detection of reload is based on the check that the worker pid did change or not
func (b *BaseSuite) haproxyReloadHappened(oldPid string) (bool, string, error) {
	newPid, err := truntime.GetGlobalHAProxyInfo(b.test.RuntimeSocketPath)
	if err != nil {
		b.T().Log(err)
		return false, "", err
	}
	fmt.Printf("oldPid/newPid: %s/%s\n", oldPid, newPid.Pid)
	return newPid.Pid != oldPid, newPid.Pid, nil
}

// ConsistentlyNoReload executes a check repeatedly for a duration.
// It t.Fatalf() if the condition ever returns false.
func (b *BaseSuite) ConsistentlyNoReload(oldPid string, duration time.Duration) {
	fmt.Printf("\n...check consistent no reload during a window: %v\n", duration)
	t := b.T()
	t.Helper()

	deadline := time.After(duration)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			// The entire duration passed without the condition failing
			return
		case <-ticker.C:
			// Check the condition
			reloadHappened, _, err := b.haproxyReloadHappened(oldPid)
			if err == nil {
				if reloadHappened {
					t.Fatal("FAIL: some reload happened")
					return
				}
			} else {
				t.Error("FAILED to get pid")
			}
		}
	}
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
