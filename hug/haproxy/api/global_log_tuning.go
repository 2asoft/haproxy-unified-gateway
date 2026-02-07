package api

import (
	"errors"

	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/haproxy-unified-gateway/hug/reload"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/diffs"
)

func (c *clientNative) UpdateGlobalLogTuning(tuning diffs.GlobalLogTuning) error {
	if c.activeTransaction == "" {
		return errors.New("global log tuning update requires an active transaction")
	}

	configuration, err := c.nativeAPI.Configuration()
	if err != nil {
		return err
	}

	version, global, err := configuration.GetGlobalConfiguration(c.activeTransaction)
	if err != nil {
		return err
	}
	if global == nil {
		global = &models.Global{}
	}

	changed, err := applyGlobalLogTuningToModel(global, tuning)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	if err = configuration.PushGlobalConfiguration(global, c.activeTransaction, version); err != nil {
		return err
	}

	reload.Instance().SetReload("Global log tuning updated")
	return nil
}

func applyGlobalLogTuningToModel(global *models.Global, tuning diffs.GlobalLogTuning) (bool, error) {
	changed, err := applyLogLineLengthToTargets(global.LogTargetList, tuning.LogLineLength)
	if err != nil {
		return false, err
	}
	tuneChanged := applyHTTPLogURILen(global, tuning.HTTPLogURILen)
	return changed || tuneChanged, nil
}

func applyLogLineLengthToTargets(targets models.LogTargets, lineLength *diffs.LogLineLength) (bool, error) {
	if lineLength != nil {
		hasTarget := false
		for _, target := range targets {
			if target != nil {
				hasTarget = true
				break
			}
		}
		if !hasTarget {
			return false, errors.New("global log line length requires at least one global log target")
		}
	}

	desiredLength := int64(0)
	if lineLength != nil {
		desiredLength = int64(*lineLength)
	}

	changed := false
	for _, target := range targets {
		if target == nil {
			continue
		}
		if target.Length == desiredLength {
			continue
		}
		target.Length = desiredLength
		changed = true
	}

	return changed, nil
}

func applyHTTPLogURILen(global *models.Global, value *diffs.LogLineLength) bool {
	desired := int64(0)
	if value != nil {
		desired = int64(*value)
	}

	if global.TuneOptions == nil {
		if desired == 0 {
			return false
		}
		global.TuneOptions = &models.TuneOptions{HTTPLogurilen: desired}
		return true
	}

	if global.TuneOptions.HTTPLogurilen == desired {
		return false
	}

	global.TuneOptions.HTTPLogurilen = desired
	return true
}
