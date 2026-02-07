package haproxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	v3 "github.com/haproxytech/haproxy-unified-gateway/api/gate/v3"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/logging"
)

type captureRequestHeader struct {
	Name   string
	Length int64
}

type frontendLogFormatConfig struct {
	logFormat             string
	captureHeaders        []captureRequestHeader
	logRequestHeaderNames bool
}

const (
	requestHeaderNamesVarScope       = "txn"
	requestHeaderNamesVarName        = "req_header_names"
	requestHeaderNamesVar            = requestHeaderNamesVarScope + "." + requestHeaderNamesVarName
	requestHeaderNamesLogFormatToken = "%[var(" + requestHeaderNamesVar + ")]"
	requestHeaderNamesSample         = "req.hdr_names"
)

func (b *HaproxyConfMgrImpl) reconcileFrontendLogFormat() bool {
	conf := b.controllerStore.ClusterStore.HugConfs[b.params.ControllerConfNsName]
	if conf == nil || conf.Spec.HaproxyDefaults == nil {
		changed := b.configuration.frontendLogFormat != "" || len(b.configuration.frontendCaptureHeaders) != 0 || b.configuration.frontendLogRequestHeaderNames
		b.configuration.frontendLogFormat = ""
		b.configuration.frontendCaptureHeaders = nil
		b.configuration.frontendLogRequestHeaderNames = false
		return changed
	}

	parsed, err := parseFrontendLogFormatConfig(conf.Spec.HaproxyDefaults)
	if err != nil {
		b.logger.LogAttrs(context.Background(), slog.LevelError, "failed to parse frontend log format",
			logging.LogAttrError(err),
		)
		return false
	}

	changed := parsed.logFormat != b.configuration.frontendLogFormat ||
		!slices.Equal(parsed.captureHeaders, b.configuration.frontendCaptureHeaders) ||
		parsed.logRequestHeaderNames != b.configuration.frontendLogRequestHeaderNames
	b.configuration.frontendLogFormat = parsed.logFormat
	b.configuration.frontendCaptureHeaders = parsed.captureHeaders
	b.configuration.frontendLogRequestHeaderNames = parsed.logRequestHeaderNames
	return changed
}

func parseFrontendLogFormatConfig(defaults *v3.HaproxyDefaults) (frontendLogFormatConfig, error) {
	logFormat := strings.TrimSpace(defaults.LogFormat)
	captureHeaders, err := parseCaptureRequestHeaders(defaults.CaptureRequestHeaders)
	if err != nil {
		return frontendLogFormatConfig{}, err
	}

	logRequestHeaderNames := defaults.LogRequestHeaderNames
	if logRequestHeaderNames {
		logFormat, err = appendRequestHeaderNamesToLogFormat(logFormat)
		if err != nil {
			return frontendLogFormatConfig{}, err
		}
	}

	return frontendLogFormatConfig{
		logFormat:             logFormat,
		captureHeaders:        captureHeaders,
		logRequestHeaderNames: logRequestHeaderNames,
	}, nil
}

func appendRequestHeaderNamesToLogFormat(logFormat string) (string, error) {
	trimmed := strings.TrimSpace(logFormat)
	if trimmed == "" {
		return "", errors.New("logFormat is required when logRequestHeaderNames is enabled")
	}
	if strings.Contains(trimmed, requestHeaderNamesVar) {
		return trimmed, nil
	}
	if strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'") && len(trimmed) > 1 {
		trimmed = strings.TrimSuffix(trimmed, "'")
		return trimmed + " headers=" + requestHeaderNamesLogFormatToken + "'", nil
	}
	return trimmed + " headers=" + requestHeaderNamesLogFormatToken, nil
}

func parseCaptureRequestHeaders(rawHeaders []v3.CaptureRequestHeader) ([]captureRequestHeader, error) {
	if len(rawHeaders) == 0 {
		return nil, nil
	}

	parsed := make([]captureRequestHeader, 0, len(rawHeaders))
	for _, header := range rawHeaders {
		name := strings.TrimSpace(header.Name)
		if name == "" {
			return nil, errors.New("capture request header name is empty")
		}
		if header.Length <= 0 {
			return nil, fmt.Errorf("capture request header length must be positive for %s", name)
		}
		parsed = append(parsed, captureRequestHeader{Name: name, Length: header.Length})
	}
	return parsed, nil
}
