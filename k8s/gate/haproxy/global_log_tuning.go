package haproxy

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/go-cmp/cmp"
	v3 "github.com/haproxytech/haproxy-unified-gateway/api/gate/v3"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/diffs"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/logging"
)

func (b *HaproxyConfMgrImpl) reconcileGlobalLogTuning() bool {
	conf := b.controllerStore.ClusterStore.HugConfs[b.params.ControllerConfNsName]
	if conf == nil {
		return b.updateGlobalLogTuning(diffs.GlobalLogTuning{})
	}

	parsed, err := parseGlobalLogTuningConfig(conf.Spec)
	if err != nil {
		b.logger.LogAttrs(context.Background(), slog.LevelError, "failed to parse global log tuning",
			logging.LogAttrError(err),
		)
		return false
	}

	return b.updateGlobalLogTuning(parsed)
}

func (b *HaproxyConfMgrImpl) updateGlobalLogTuning(parsed diffs.GlobalLogTuning) bool {
	changed := !cmp.Equal(parsed, b.configuration.globalLogTuning)
	b.configuration.globalLogTuning = parsed
	if !changed {
		return false
	}
	b.configuration.diffs.GlobalLogTuning = &b.configuration.globalLogTuning
	return true
}

func parseGlobalLogTuningConfig(spec v3.ControllerConfSpec) (diffs.GlobalLogTuning, error) {
	if spec.HaproxyGlobal == nil {
		return diffs.GlobalLogTuning{}, nil
	}

	logLineLength, err := parseOptionalLogLineLength(spec.HaproxyGlobal.LogLineLength, "logLineLength")
	if err != nil {
		return diffs.GlobalLogTuning{}, err
	}
	httpLogURILen, err := parseOptionalLogLineLength(spec.HaproxyGlobal.HTTPLogURILen, "httpLogUriLen")
	if err != nil {
		return diffs.GlobalLogTuning{}, err
	}
	return diffs.GlobalLogTuning{
		LogLineLength: logLineLength,
		HTTPLogURILen: httpLogURILen,
	}, nil
}

func parseOptionalLogLineLength(value *int64, fieldName string) (*diffs.LogLineLength, error) {
	if value == nil {
		return nil, nil
	}
	parsed, err := parseLogLineLength(*value, fieldName)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseLogLineLength(value int64, fieldName string) (diffs.LogLineLength, error) {
	if value <= 0 {
		return 0, fmt.Errorf("%s must be positive", fieldName)
	}
	return diffs.LogLineLength(value), nil
}
