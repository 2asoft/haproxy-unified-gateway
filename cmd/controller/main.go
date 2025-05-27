package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/haproxytech/kubernetes-controller/cmd/controller/version"
	"github.com/joho/godotenv"
	"k8s.io/apimachinery/pkg/types"

	controller "github.com/haproxytech/kubernetes-controller/k8s/gate"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/config"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	opt "github.com/haproxytech/kubernetes-controller/k8s/gate/options"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
)

//revive:disable

// var testKubeConfig = `
// apiVersion: v1
// clusters:
// - cluster:
//     certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCVENDQWUyZ0F3SUJBZ0lJTnZTMUMzMUdKVkV3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TlRBME1qa3hNREF3TWpSYUZ3MHpOVEEwTWpjeE1EQTFNalJhTUJVeApFekFSQmdOVkJBTVRDbXQxWW1WeWJtVjBaWE13Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLCkFvSUJBUUNjaUp1K1lwK2o5NzQxWHpvczZnelhReEFzUkZEaHVJTjJFZDl6UDQ0SFBIV3RnMFFOYjN5QzRLVkcKVFowb29mTXEzelhiYVZacHJsM3VHQitESWdtTmFmZWk2TENLblN4Mnl4RFYxWkhKd25lUU5oSUVybXVzZHI3UApqK3poZjZTU2drVFhnMnhJUlkyMWYwL2lKMFFqcTJVYmJrZlNDVnlGWDN1ZGVadmtaU3J5L1B2Z2JkdE8xMmZqClFKRE92eG4yRTI4aHNGaVQ2MXl2ZW16RDZnZ0ZLSzEzSEVMejVteWR5THEzNzFlWDZtOWJrajRycUZjdktnSi8KRUNXUEg4TWlzQ1FpMmZGT0R6ME5ETlhYTDFmU2Y3Uzk1UG5IblVGWnA3U3BzN0V1bldZeElQS1Q0V3JqaDhRcQp5VU8rR2JRK1pqVVFqMWZiZmphdFJFbGN3UGZQQWdNQkFBR2pXVEJYTUE0R0ExVWREd0VCL3dRRUF3SUNwREFQCkJnTlZIUk1CQWY4RUJUQURBUUgvTUIwR0ExVWREZ1FXQkJUanBRMlp3QVY4RC9lWlRJVkR5VnBEeFBJWmhEQVYKQmdOVkhSRUVEakFNZ2dwcmRXSmxjbTVsZEdWek1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQXBpcVM5MXRDQQpzSFBTUSs2K05qRTBQVVU4RUZqcDdtU3FtRlFwSHdMNm5FUTMxNGtlaWlMWm9ZUEZBNHZvVGJ1aGM0eWZFNzhhCjdDUlNzMGtadkFQcmNneGYyZzhEOWh5aExkN1JLVVNocUlKQmNOUWdUeFgzK1ExenJveCtLQW9sSE5zWjN2RkkKSjZJbHVnaDJ5Sk14SlN5Znl6MHUycVdZUHFVaDNzQ1Y0b1V6QXB4aXlZeWRFUmFTcS9QSFF5WSs4ZithMWdrdwpIQkxTbDFHOWk4YTlHM3ZVRXdIQit5enBQVjZubjE5L1YzdzQ2M2s3U2xYNnR6UDdFQ1F5djBrQmcyaUQ3d3ViCkdabXYzS3h0cTIrYUVpS285cHpYdGFPdk0yUlVpN2NnVUF0aHlSZEV2Y2FRUUUxMkNHRzlkdTNkNzR3TXlFd2gKVGZYVjI0SDF4dTFSCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
//     server: https://127.0.0.1:6443
//   name: kind-dev
// contexts:
// - context:
//     cluster: kind-dev
//     user: kind-dev
//   name: kind-dev
// current-context: kind-dev
// kind: Config
// preferences: {}
// users:
// - name: kind-dev
//   user:
//     client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURLVENDQWhHZ0F3SUJBZ0lJRVA5VVJ3YUcyNTh3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TlRBME1qa3hNREF3TWpSYUZ3MHlOakEwTWpreE1EQTFNalZhTUR3eApIekFkQmdOVkJBb1RGbXQxWW1WaFpHMDZZMngxYzNSbGNpMWhaRzFwYm5NeEdUQVhCZ05WQkFNVEVHdDFZbVZ5CmJtVjBaWE10WVdSdGFXNHdnZ0VpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFEU0ViZGcKa3pHWDRwTmNIQkhhbHU5MXVXK2ZobkNUN08yS2NKVDZoNDRhTDF0UkxpSTZMQ00wcXBNakx3Zkg0UFdDTjY0SwphNkd5QVFXWGEwSjE3TUsrKzJFZTFGVWFMbUhuVDRRVmV6ZzBodmhnNjZ2cW95R1pXdExZSHpuVTRkc1lST3Y4CnE2c01tRTlUYlZNZnFUdERGYmlGcDNWZlN3emJ1b2dET252S2pCYW9wV1BYVHl4UGtVZDljd0tnMDhDSFVPQWcKaW5pUkhjVkNUa1BWZUxxVHlwRXRud1REUWVwQjVNYmJtY3dGUUhjWHh6alo4ekN2NzMwUmpNUWVnVlpVYTQ2NQpkZWt2eUMvQTVWYjV0cm1kNERISXEvY2ZyazNQeWFsdXVTeDZzWDJNR2diT1NMVDBwK0tuOFI4STd0WWE0UW55Cmg4dWp2ZTcvU1ZCcmNWd1JBZ01CQUFHalZqQlVNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUsKQmdnckJnRUZCUWNEQWpBTUJnTlZIUk1CQWY4RUFqQUFNQjhHQTFVZEl3UVlNQmFBRk9PbERabkFCWHdQOTVsTQpoVVBKV2tQRThobUVNQTBHQ1NxR1NJYjNEUUVCQ3dVQUE0SUJBUUFSalR3enVpUUQzN3lVRXkyK3gwRDR5aTN3CjBSZnl3YTB0bTlkTUJUVzl5VW1SSWN0dGlwSHhja1FUc05VMWJFYVRxRUp4NkQ5SVhnQVBxRDByK2kzNEtjME0KSmdyNHphRGhZaEtLWWhyWXJGMUJ3QmpiNFpIRnZ1Mm9hRnRlbEFqcWJ0N2M4QTIraGRzUUtFSVZHQnQrM2RYbwphV3h4UlVKVTdGNlhJZXAwK1VaMEVrOGVyeWIreGEzMkIxeVB1Q3NVNVFPdVFmUlY0ZWdpY0NNdVZlSWpJaVM2CnR5TjZPdTltMnJVVm4wckdTVEZObUNDU2JQRXE5OHliRnVWVEliZWQwL2dzUHBnZFF5Z0JmT0t4QWw3TC8xb00KQUwxZkp1M2IrcCtxT1JlQm85NnYvTlQ3bTJhakMrc2Eyb3cvQ01oYWZLekJLdjdYUXc5SWFSRUo4WXlOCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
//     client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBMGhHM1lKTXhsK0tUWEJ3UjJwYnZkYmx2bjRad2srenRpbkNVK29lT0dpOWJVUzRpCk9pd2pOS3FUSXk4SHgrRDFnamV1Q211aHNnRUZsMnRDZGV6Q3Z2dGhIdFJWR2k1aDUwK0VGWHM0TkliNFlPdXIKNnFNaG1WclMyQjg1MU9IYkdFVHIvS3VyREpoUFUyMVRINms3UXhXNGhhZDFYMHNNMjdxSUF6cDd5b3dXcUtWagoxMDhzVDVGSGZYTUNvTlBBaDFEZ0lJcDRrUjNGUWs1RDFYaTZrOHFSTFo4RXcwSHFRZVRHMjVuTUJVQjNGOGM0CjJmTXdyKzk5RVl6RUhvRldWR3VPdVhYcEw4Z3Z3T1ZXK2JhNW5lQXh5S3YzSDY1Tno4bXBicmtzZXJGOWpCb0cKemtpMDlLZmlwL0VmQ083V0d1RUo4b2ZMbzczdS8wbFFhM0ZjRVFJREFRQUJBb0lCQUZMRXhEbjdCUWxSTHJxVwpITHJCeWF2YTJvNUNURTBjaHlPSzVFZ3A3T1dJVHpTWE5za3c1dFl6ZHpIZnIvTWpRZGlDMDhJclVsUnVicU9RCmtXa2hWa0lsamNpMTVLb2lLRlVaVVhPZFR6SHpGQjRyL1ZxLzE5Y3luK3lqc1FlZHpkT3NKRWN6NUh0Yjc3VngKVjlVYnVzdmQzUXhjUkxTOVAxMjhDeWNxZmVmNXZyNE9Oa1V1VG1EZTQ2cnNCR1hoekh2VFRVdGNPejdmb291dwpPM09lTE1yK2V5RVJxeW9TZ2NWWTRGcmNYR3BPbW5neENDbWU4OWJPN0dtY0I1Qk9EVTNqNkNIWVQ1QTZUUGpzCkNzSlNYcVVjbUpHWlFuV1ZVcnF6SmxUQUZoQUZPbzN4Q29TSGI1ZkZGdmRsZ25rT0dxWXlPM3h3NGRONW5KLzMKbUVHMDBSa0NnWUVBMGxCaUJseDl5OHc2UUtuUlRuMm9saXV0SHppdm50TG9YRGg5QXEvdjl1R0h5N0llL2Z1MgpzamJXT0ZleEVjSjlHRVZUKzFSZFJlQTd5aW9nSGU2WUFlVzF0d1hGZXpaWXdtMDVockxFOWFYdDB1aDJyWnlmCjhnTHprYnBWTXNESG9TMDJKaU1ZQlQ1V1BKRUc5ZDdsTDM0Y1E3TWNDY1FXNGVEVXVWMi9BbE1DZ1lFQS83TzQKY2xNM0xiakRsSFhaSENnR1NCcVE3UFVCYmhEU3lwUko3OEVDZkJyLzh4eDU1YlYyd2djdkdQVVVuVDdvQzJyYQpnRlg4dXlXbUpPajNkQ2hjV3pIRVJ2V25HTEJGWmthWUtzbmdyeGhCRWtJZTY2aExtcmJlTmoybjhDRGVLQlVZCkdhT1RvSTAxYUFYYm03bWdjalF2enpRT1NRU2pMd3lIV0YwM1k0c0NnWUE1c0pVQys3SUNDalpjY0hpYW1EdDcKWGVXeUw4RjB4cE80WUVKaVQxSjZuU2k3eGxOY0JnVDZZN0psYUNDSko1bGE1QUdDYW9UZld2L3JsNXlSdVZYMwpCMFRPUElZTUl6ODdyZXhldDREeGhSOTBnQkcxMDhYSUEraytLeWVkc1dYUkgyN0FEVlpVY2VJRDRTQlFwMkNrCm8yb3JZK0VvQ0tMaU9PTUJLZWJ3UXdLQmdFbE4ySDdONUcrekhENmZXbEo4RnZEc3pNZGhwYnRNRDJJTUNQWTIKdXVPaFNlY0VMdDN2bTlBY0J5QjhnaUJpUEZ1cGttSmdSRWZTajBMZGxyTXlMdWZsNklML1Fad09USmI1ZmY0bQpTY2RvaUo4WFhZM3BmV01wTWFNVEllWHhSajd2YlMxTWU3SDNTV3c4NGF4UEZ2UW1pZDQ0Nmk5OHFOdUFGL3o1CkhEdnBBb0dBV2RJR3NnaGVrVlp1Q1pVMW1zUnovYTlGdnRYNWhhOVVMNERFZGNwU3RWbTR2V3pDUkZ0b2gydTEKUndkb21LWWlweG9hY0pNTGh1Z3NiaWZYQmdxMHBqRlNJd0cyL1FnRGIrOUZiRlVHTnJBZ0FFeVY3SWhxbS94ZAptRUx5NE5lTGRjUkZnY1dUTG1CSld5ZllmQjFsWXFVNXBBcGRRdDBVbGNibUZIZWt3RzQ9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==

// `

//revive:enable

func main() {
	_ = godotenv.Load()
	fmt.Println(string(version.Info))
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	controllerConfig, err := createControllerPodConfig()
	if err != nil {
		panic(fmt.Errorf("error creating controller pod config: %w", err))
	}
	// Values to get from flags
	// to implement:  flags
	metricsConfig := config.MetricsConfig{
		Port:    6062,
		Enabled: false,
		Secure:  false,
	}
	// if gatewayClass =is empty, we will support all GatewayClasses that reference this controller
	// (through the spec.controllerName)
	gatewayClasses := []string{"haproxy"}
	controllerName := "gate.haproxy.org/gateway-controller"
	controllerConfName := types.NamespacedName{
		Namespace: "test",
		Name:      "haproxyctrlconf",
	}

	whiteListNs := []string{"default", "kube-system", "haproxy-controller", "test", "test2"}
	// whiteListNs := []string{}

	// kubeconfig := testKubeConfig
	kubeconfig := ""

	syncPeriod := 1 * time.Second
	logLevel := slog.LevelDebug
	logCategories := []string{"all"}

	leaderElectionLockName := "kubernetes-controller-leader-election-lock"
	leaderElectionConfig := config.LeaderElectionConfig{
		Enabled:  false,
		LockName: leaderElectionLockName,
		Identity: controllerConfig.Name,
	}

	treeCh := make(chan *tree.GateTree, 100)

	cntlr, err := controller.New(
		opt.ControllerPodConfig(controllerConfig),
		opt.KubeConfig(kubeconfig),
		opt.GatewayClass(gatewayClasses),
		opt.ControllerConf(controllerConfName),
		opt.SyncPeriod(syncPeriod),
		opt.MetricsConfig(metricsConfig),
		opt.LeaderElectionConfig(leaderElectionConfig),
		opt.ControllerName(controllerName),
		opt.WhiteListNamespaces(whiteListNs),
		opt.Logging(logLevel, logCategories),
		opt.TreeChannel(treeCh),
	)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	go func() {
		err := cntlr.Run(ctx, &wg)
		if err != nil {
			panic(err)
		}
	}()

	// Goroutine to listen on treeCh and print received GateTree objects
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				cntlr.Configuration.Logger.LogAttrs(context.Background(), slog.LevelInfo,
					"shutting down tree handler goroutine",
					logging.LogAttrCategory(logging.LogCategoryGate),
				)
				return
			case gt := <-treeCh:
				cntlr.Configuration.Logger.LogAttrs(context.Background(), slog.LevelDebug,
					"received new GateTree",
					logging.LogAttrCategory(logging.LogCategoryGate),
					slog.String("tree", fmt.Sprintf("Received new GateTree: %+v", gt.GatewayClasses)))
			}
		}
	}()

	<-ctx.Done()
	cntlr.Configuration.Logger.Info("shutting down controller")
	wg.Wait()
}

func createControllerPodConfig() (config.ControllerPodConfig, error) {
	podIP, err := getValueFromEnv("POD_IP")
	if err != nil {
		return config.ControllerPodConfig{}, err
	}

	ns, err := getValueFromEnv("POD_NAMESPACE")
	if err != nil {
		return config.ControllerPodConfig{}, err
	}

	name, err := getValueFromEnv("POD_NAME")
	if err != nil {
		return config.ControllerPodConfig{}, err
	}

	c := config.ControllerPodConfig{
		PodIP:     podIP,
		Namespace: ns,
		Name:      name,
	}

	return c, nil
}

func getValueFromEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("environment variable %s not set", key)
	}

	return val, nil
}
