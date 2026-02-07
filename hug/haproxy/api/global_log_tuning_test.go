package api

import (
	"testing"

	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/diffs"
)

func TestApplyGlobalLogTuningToModelUpdatesGlobalLogTargetsAndTuneOptions(t *testing.T) {
	logLineLength := diffs.LogLineLength(4096)
	httpLogURILen := diffs.LogLineLength(8192)

	global := models.Global{
		GlobalBase: models.GlobalBase{
			TuneOptions: &models.TuneOptions{},
		},
		LogTargetList: models.LogTargets{
			&models.LogTarget{Address: "stdout", Facility: "daemon"},
			&models.LogTarget{Address: "stdout", Facility: "local0"},
		},
	}

	changed, err := applyGlobalLogTuningToModel(&global, diffs.GlobalLogTuning{
		LogLineLength: &logLineLength,
		HTTPLogURILen: &httpLogURILen,
	})
	if err != nil {
		t.Fatalf("applyGlobalLogTuningToModel returned error: %v", err)
	}
	if !changed {
		t.Fatal("expected model to be changed")
	}
	for i, target := range global.LogTargetList {
		if target == nil {
			t.Fatalf("expected log target %d to be non-nil", i)
		}
		if target.Length != int64(logLineLength) {
			t.Fatalf("expected log target %d length %d, got %d", i, logLineLength, target.Length)
		}
	}
	if global.TuneOptions == nil {
		t.Fatal("expected tune options to be present")
	}
	if global.TuneOptions.HTTPLogurilen != int64(httpLogURILen) {
		t.Fatalf("expected tune.http.logurilen=%d, got %d", httpLogURILen, global.TuneOptions.HTTPLogurilen)
	}
}

func TestApplyGlobalLogTuningToModelClearsConfiguredValuesWhenUnset(t *testing.T) {
	global := models.Global{
		GlobalBase: models.GlobalBase{
			TuneOptions: &models.TuneOptions{HTTPLogurilen: 4096},
		},
		LogTargetList: models.LogTargets{
			&models.LogTarget{Address: "stdout", Facility: "daemon", Length: 4096},
		},
	}

	changed, err := applyGlobalLogTuningToModel(&global, diffs.GlobalLogTuning{})
	if err != nil {
		t.Fatalf("applyGlobalLogTuningToModel returned error: %v", err)
	}
	if !changed {
		t.Fatal("expected model to be changed")
	}
	if global.LogTargetList[0].Length != 0 {
		t.Fatalf("expected cleared log target length, got %d", global.LogTargetList[0].Length)
	}
	if global.TuneOptions == nil {
		t.Fatal("expected tune options to be present")
	}
	if global.TuneOptions.HTTPLogurilen != 0 {
		t.Fatalf("expected cleared tune.http.logurilen, got %d", global.TuneOptions.HTTPLogurilen)
	}
}

func TestApplyGlobalLogTuningToModelReturnsErrorWhenLogLineLengthHasNoTargets(t *testing.T) {
	logLineLength := diffs.LogLineLength(4096)
	global := models.Global{}

	changed, err := applyGlobalLogTuningToModel(&global, diffs.GlobalLogTuning{LogLineLength: &logLineLength})
	if err == nil {
		t.Fatal("expected error when no global log targets are configured")
	}
	if changed {
		t.Fatal("expected unchanged model when update fails")
	}
}
