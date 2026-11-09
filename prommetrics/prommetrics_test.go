// SPDX-License-Identifier: BSD-3-Clause

package prommetrics_test

import (
	"testing"

	. "github.com/nbgrp/pkg/prommetrics"
	"github.com/nbgrp/pkg/prommetrics/testutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRegistry_OwnRegisterer checks that the registry without an explicit
// registerer uses its own one, so its metrics are not registered in the default
// one.
func TestNewRegistry_OwnRegisterer(t *testing.T) {
	t.Parallel()

	NewRegistry().NewCounter(prometheus.CounterOpts{
		Name: "own_registerer_total",
		Help: "Total.",
	})

	assert.NotContains(t, testutil.GatherFamilies(t, prometheus.DefaultGatherer), "own_registerer_total")
}

// TestNewRegistry_WithRegisterer checks that the metrics of the registry are
// registered in the given registerer.
func TestNewRegistry_WithRegisterer(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	NewRegistry(WithRegisterer(reg)).NewCounter(prometheus.CounterOpts{
		Name: "given_registerer_total",
		Help: "Total.",
	})

	assert.Contains(t, testutil.GatherFamilies(t, reg), "given_registerer_total")
	assert.NotContains(t, testutil.GatherFamilies(t, prometheus.DefaultGatherer), "given_registerer_total")
}

// TestNewRegistry_WithRegisterer_Nil checks that the registry with a nil
// registerer uses its own one, just like the registry without the option.
func TestNewRegistry_WithRegisterer_Nil(t *testing.T) {
	t.Parallel()

	NewRegistry(WithRegisterer(nil)).NewCounter(prometheus.CounterOpts{
		Name: "nil_registerer_total",
		Help: "Total.",
	})

	assert.NotContains(t, testutil.GatherFamilies(t, prometheus.DefaultGatherer), "nil_registerer_total")
}

// TestNewRegistry_Namespace checks that the namespace of the registry is
// applied to the metrics.
func TestNewRegistry_Namespace(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	NewRegistry(WithRegisterer(reg), WithNamespace("app")).
		NewCounter(prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total requests.",
		})

	assert.Contains(t, testutil.GatherFamilies(t, reg), "app_requests_total")
}

// TestNewRegistry_Namespace_Overridden checks that the namespace of the
// options takes priority over the namespace of the registry.
func TestNewRegistry_Namespace_Overridden(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	NewRegistry(WithRegisterer(reg), WithNamespace("app")).
		NewCounter(prometheus.CounterOpts{
			Namespace: "worker",
			Name:      "requests_total",
			Help:      "Total requests.",
		})

	assert.Contains(t, testutil.GatherFamilies(t, reg), "worker_requests_total")
	assert.NotContains(t, testutil.GatherFamilies(t, reg), "app_requests_total")
}

// TestNewMetrics checks that every metric constructor of the registry takes
// the namespace from the registry, gives the priority to the namespace of the
// options and registers the metric in the registerer of the registry.
func TestNewMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// family is the name of the metric family without a namespace.
		family string
		// create creates the metric with the given namespace in its options.
		create func(r *Registry, namespace string) prometheus.Collector
		// labels are the expected labels of the metric.
		labels []string
	}{
		{
			name:   "counter",
			family: "requests_total",
			create: func(r *Registry, namespace string) prometheus.Collector {
				return r.NewCounter(prometheus.CounterOpts{
					Namespace: namespace,
					Name:      "requests_total",
					Help:      "Total requests.",
				})
			},
		},
		{
			name:   "counter_vec",
			family: "requests_by_code_total",
			labels: []string{"code=200"},
			create: func(r *Registry, namespace string) prometheus.Collector {
				vec := r.NewCounterVec(prometheus.CounterOpts{
					Namespace: namespace,
					Name:      "requests_by_code_total",
					Help:      "Total requests by code.",
				}, []string{"code"})
				vec.WithLabelValues("200")

				return vec
			},
		},
		{
			name:   "gauge",
			family: "workers",
			create: func(r *Registry, namespace string) prometheus.Collector {
				return r.NewGauge(prometheus.GaugeOpts{
					Namespace: namespace,
					Name:      "workers",
					Help:      "Number of workers.",
				})
			},
		},
		{
			name:   "gauge_vec",
			family: "workers_by_queue",
			labels: []string{"queue=main"},
			create: func(r *Registry, namespace string) prometheus.Collector {
				vec := r.NewGaugeVec(prometheus.GaugeOpts{
					Namespace: namespace,
					Name:      "workers_by_queue",
					Help:      "Number of workers by queue.",
				}, []string{"queue"})
				vec.WithLabelValues("main")

				return vec
			},
		},
		{
			name:   "summary",
			family: "request_duration_seconds",
			create: func(r *Registry, namespace string) prometheus.Collector {
				return r.NewSummary(prometheus.SummaryOpts{
					Namespace: namespace,
					Name:      "request_duration_seconds",
					Help:      "Request duration.",
				})
			},
		},
		{
			name:   "summary_vec",
			family: "request_duration_by_code_seconds",
			labels: []string{"code=200"},
			create: func(r *Registry, namespace string) prometheus.Collector {
				vec := r.NewSummaryVec(prometheus.SummaryOpts{
					Namespace: namespace,
					Name:      "request_duration_by_code_seconds",
					Help:      "Request duration by code.",
				}, []string{"code"})
				vec.WithLabelValues("200")

				return vec
			},
		},
		{
			name:   "histogram",
			family: "request_size_bytes",
			create: func(r *Registry, namespace string) prometheus.Collector {
				return r.NewHistogram(prometheus.HistogramOpts{
					Namespace: namespace,
					Name:      "request_size_bytes",
					Help:      "Request size.",
				})
			},
		},
		{
			name:   "histogram_vec",
			family: "request_size_by_code_bytes",
			labels: []string{"code=200"},
			create: func(r *Registry, namespace string) prometheus.Collector {
				vec := r.NewHistogramVec(prometheus.HistogramOpts{
					Namespace: namespace,
					Name:      "request_size_by_code_bytes",
					Help:      "Request size by code.",
				}, []string{"code"})
				vec.WithLabelValues("200")

				return vec
			},
		},
	}

	scenarios := []struct {
		name string
		// registryNamespace is the namespace of the registry.
		registryNamespace string
		// optsNamespace is the namespace set in the options of the metric.
		optsNamespace string
		// wantPrefix is the expected prefix of the name of the metric family.
		wantPrefix string
	}{
		{
			name:              "namespace of the registry",
			registryNamespace: "app",
			wantPrefix:        "app_",
		},
		{
			name:              "namespace of the options",
			registryNamespace: "app",
			optsNamespace:     "worker",
			wantPrefix:        "worker_",
		},
		{
			name: "without namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, sc := range scenarios {
				t.Run(sc.name, func(t *testing.T) {
					t.Parallel()

					reg := prometheus.NewRegistry()
					opts := []Option{WithRegisterer(reg)}
					if sc.registryNamespace != "" {
						opts = append(opts, WithNamespace(sc.registryNamespace))
					}

					tt.create(NewRegistry(opts...), sc.optsNamespace)

					families := testutil.GatherFamilies(t, reg)
					want := sc.wantPrefix + tt.family
					require.Contains(t, families, want)
					assert.ElementsMatch(t, tt.labels, families[want])

					// The metric is registered in the registerer of the registry only.
					assert.NotContains(t, testutil.GatherFamilies(t, prometheus.DefaultGatherer), want)
				})
			}
		})
	}
}

// TestRegistry_Register checks that the registration of the collectors is
// delegated to the registerer of the registry.
func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	r := NewRegistry(WithRegisterer(reg))

	c := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "registered_total",
		Help: "Total.",
	})

	require.NoError(t, r.Register(c))
	assert.Contains(t, testutil.GatherFamilies(t, reg), "registered_total")

	// The collector cannot be registered twice.
	require.Error(t, r.Register(c))

	assert.True(t, r.Unregister(c))
	assert.NotContains(t, testutil.GatherFamilies(t, reg), "registered_total")

	// The unregistered collector cannot be unregistered again.
	assert.False(t, r.Unregister(c))
}

// TestRegistry_MustRegister checks that the registry panics if the collector
// cannot be registered.
func TestRegistry_MustRegister(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	r := NewRegistry(WithRegisterer(reg))

	c := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "must_registered_total",
		Help: "Total.",
	})

	r.MustRegister(c)
	assert.Contains(t, testutil.GatherFamilies(t, reg), "must_registered_total")

	assert.Panics(t, func() {
		r.MustRegister(c)
	})
}

// TestRegistry_Registerer checks that the registry implements the
// prometheus.Registerer interface, so it can be used wherever the registerer
// is expected.
func TestRegistry_Registerer(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	var r prometheus.Registerer = NewRegistry(WithRegisterer(reg))

	r.MustRegister(prometheus.NewCounter(prometheus.CounterOpts{
		Name: "as_registerer_total",
		Help: "Total.",
	}))

	assert.Contains(t, testutil.GatherFamilies(t, reg), "as_registerer_total")
}

// TestMustRegisterGoCollectors_Registry checks that the registry can be used
// as the registerer of the Go collectors.
func TestMustRegisterGoCollectors_Registry(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	MustRegisterGoCollectors(NewRegistry(WithRegisterer(reg)))

	families := testutil.GatherFamilies(t, reg)
	assert.Contains(t, families, "go_goroutines")
	assert.Contains(t, families, "process_cpu_seconds_total")
}
