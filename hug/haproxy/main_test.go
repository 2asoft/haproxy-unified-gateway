package haproxy

import (
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	parser "github.com/haproxytech/client-native/v6/config-parser"
	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/client-native/v6/runtime"
	"github.com/haproxytech/haproxy-unified-gateway/hug/haproxy/api"
	"github.com/haproxytech/haproxy-unified-gateway/hug/reload"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/diffs"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/structured"
)

func TestApplyCfgUpdatesStartsTransactionBeforeGlobalLogTuning(t *testing.T) {
	reload.Instance().Reset()
	defer reload.Instance().Reset()

	client := &orderingTestClient{}
	manager := AppManagerImpl{
		client:  client,
		process: &orderingTestProcess{},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	logLineLength := diffs.LogLineLength(2048)
	err := manager.applyCfgUpdates(diffs.HaproxyConfDiffs{
		Created: structured.NewStructuredConf(),
		Updated: structured.NewStructuredConf(),
		Deleted: structured.NewStructuredConf(),
		GlobalLogTuning: &diffs.GlobalLogTuning{
			LogLineLength: &logLineLength,
		},
	})
	if err != nil {
		t.Fatalf("applyCfgUpdates returned error: %v", err)
	}

	expectedCalls := []string{"start", "global", "commit", "dispose"}
	if !reflect.DeepEqual(client.calls, expectedCalls) {
		t.Fatalf("unexpected API call order: got %v, want %v", client.calls, expectedCalls)
	}
}

type orderingTestClient struct {
	started bool
	calls   []string
}

func (c *orderingTestClient) APIStartTransaction() error {
	if c.started {
		return errors.New("transaction already started")
	}
	c.started = true
	c.calls = append(c.calls, "start")
	return nil
}

func (c *orderingTestClient) APICommitTransaction() error {
	return nil
}

func (c *orderingTestClient) APIFinalCommitTransaction() error {
	if !c.started {
		return errors.New("transaction not started")
	}
	c.calls = append(c.calls, "commit")
	return nil
}

func (c *orderingTestClient) APIDisposeTransaction() {
	c.calls = append(c.calls, "dispose")
	c.started = false
}

func (c *orderingTestClient) FrontendCreate(frontend models.Frontend) error {
	return nil
}

func (c *orderingTestClient) FrontendDelete(frontendName string) error {
	return nil
}

func (c *orderingTestClient) FrontendsGet() (models.Frontends, error) {
	return nil, nil
}

func (c *orderingTestClient) FrontendGet(frontendName string) (models.Frontend, error) {
	return models.Frontend{}, nil
}

func (c *orderingTestClient) FrontendEdit(frontend models.Frontend) error {
	return nil
}

func (c *orderingTestClient) BindsGet(parentType parser.Section, name string) (models.Binds, error) {
	return nil, nil
}

func (c *orderingTestClient) BindCreate(parentType parser.Section, name string, bind models.Bind) error {
	return nil
}

func (c *orderingTestClient) BindEdit(parentType parser.Section, name string, bind models.Bind) error {
	return nil
}

func (c *orderingTestClient) BindDelete(parentType parser.Section, name string, bind string) error {
	return nil
}

func (c *orderingTestClient) BindDeleteAll(parentType parser.Section, name string) error {
	return nil
}

func (c *orderingTestClient) BackendCreate(backend models.Backend) error {
	return nil
}

func (c *orderingTestClient) BackendDelete(backendName string) error {
	return nil
}

func (c *orderingTestClient) BackendsGet() (models.Backends, error) {
	return nil, nil
}

func (c *orderingTestClient) BackendGet(backendName string) (models.Backend, error) {
	return models.Backend{}, nil
}

func (c *orderingTestClient) BackendEdit(backend models.Backend) error {
	return nil
}

func (c *orderingTestClient) GlobalGet() (models.Global, error) {
	return models.Global{}, nil
}

func (c *orderingTestClient) GlobalEdit(global *models.Global, mergeStrategy string) error {
	return nil
}

func (c *orderingTestClient) DefaultsSectionGet(name string) (*models.Defaults, error) {
	return &models.Defaults{}, nil
}

func (c *orderingTestClient) DefaultsSectionEdit(defaults *models.Defaults, mergeStrategy string) error {
	return nil
}

func (c *orderingTestClient) UpdateGlobalLogTuning(tuning diffs.GlobalLogTuning) error {
	if !c.started {
		return errors.New("global log tuning update must run in a transaction")
	}
	c.calls = append(c.calls, "global")
	return nil
}

func (c *orderingTestClient) RuntimeClient() runtime.Runtime {
	return nil
}

var _ api.HAProxyClient = &orderingTestClient{}

type orderingTestProcess struct{}

func (p *orderingTestProcess) Service(action string) (string, error) {
	return "", nil
}

func (p *orderingTestProcess) UseAuxFile(useAuxFile bool) {
}

func (p *orderingTestProcess) SetAPI(client api.HAProxyClient) {
}
