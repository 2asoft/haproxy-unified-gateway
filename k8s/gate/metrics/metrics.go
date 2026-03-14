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
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	namespace = "hug"
)

var (
	// Event batch metrics
	EventBatchDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "event_batch_duration_seconds",
		Help:      "Duration of event batch processing in seconds.",
		Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})

	EventBatchSize = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "event_batch_size",
		Help:      "Number of events in a processed batch.",
		Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500},
	})

	EventBatchTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "event_batch_total",
		Help:      "Total number of event batches processed.",
	})

	EventBatchErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "event_batch_errors_total",
		Help:      "Total number of event batch processing errors.",
	})

	// HAProxy config generation metrics
	ConfigGenerationDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "config_generation_duration_seconds",
		Help:      "Duration of HAProxy configuration diff computation in seconds.",
		Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
	})

	ConfigTransferDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "config_transfer_duration_seconds",
		Help:      "Duration of HAProxy configuration transfer and application in seconds.",
		Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})

	HaproxyReloadTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "haproxy_reload_total",
		Help:      "Total number of HAProxy configuration reloads.",
	})

	// Config diff metrics
	ConfigDiffs = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "config_diffs_total",
		Help:      "Total number of HAProxy configuration diffs by operation and resource type.",
	}, []string{"operation", "resource"})

	// K8s resource event metrics
	EventsProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "events_processed_total",
		Help:      "Total number of Kubernetes resource events processed by event type.",
	}, []string{"event_type"})

	// Certificate storage metrics
	CertStorageOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "cert_storage_operations_total",
		Help:      "Total number of certificate storage operations.",
	}, []string{"operation", "status"})

	CertStorageDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "cert_storage_duration_seconds",
		Help:      "Duration of certificate storage operations in seconds.",
		Buckets:   []float64{.0001, .0005, .001, .005, .01, .05, .1},
	}, []string{"operation"})

	// Crt-list storage metrics
	CrtListStorageOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "crtlist_storage_operations_total",
		Help:      "Total number of crt-list storage operations.",
	}, []string{"operation", "status"})

	// Certificate runtime operations
	CertRuntimeOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "cert_runtime_operations_total",
		Help:      "Total number of certificate runtime operations (create/update/delete via HAProxy runtime API).",
	}, []string{"operation", "status"})

	// Maps storage metrics
	MapStorageOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "map_storage_operations_total",
		Help:      "Total number of map file storage operations.",
	}, []string{"operation", "status"})
)

func init() {
	metrics.Registry.MustRegister(
		EventBatchDuration,
		EventBatchSize,
		EventBatchTotal,
		EventBatchErrors,
		ConfigGenerationDuration,
		ConfigTransferDuration,
		HaproxyReloadTotal,
		ConfigDiffs,
		EventsProcessed,
		CertStorageOperations,
		CertStorageDuration,
		CrtListStorageOperations,
		CertRuntimeOperations,
		MapStorageOperations,
	)
}
