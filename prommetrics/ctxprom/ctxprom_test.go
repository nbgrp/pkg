// SPDX-License-Identifier: BSD-3-Clause

package ctxprom_test

import (
	"context"
	"testing"

	"github.com/nbgrp/pkg/prommetrics"
	. "github.com/nbgrp/pkg/prommetrics/ctxprom"
	"github.com/nbgrp/pkg/prommetrics/testutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToContext(t *testing.T) {
	t.Parallel()

	reg := prommetrics.NewRegistry(prommetrics.WithRegisterer(prometheus.NewRegistry()))
	ctx := ToContext(t.Context(), reg)

	assert.Same(t, reg, FromContext(ctx))
}

// TestFromContext_Default checks that the context without a registry falls
// back to the default registerer.
func TestFromContext_Default(t *testing.T) {
	t.Parallel()

	r := FromContext(t.Context())
	require.NotNil(t, r)

	r.NewCounter(prometheus.CounterOpts{
		Name: "default_registerer_total",
		Help: "Total.",
	})

	assert.Contains(t, testutil.GatherFamilies(t, prometheus.DefaultGatherer), "default_registerer_total")
}

// TestFromContext_NilRegistry checks that the context with a nil registry
// falls back to the default registerer.
func TestFromContext_NilRegistry(t *testing.T) {
	t.Parallel()

	ctx := ToContext(t.Context(), nil)

	r := FromContext(ctx)
	require.NotNil(t, r)

	r.NewCounter(prometheus.CounterOpts{
		Name: "nil_registry_total",
		Help: "Total.",
	})

	assert.Contains(t, testutil.GatherFamilies(t, prometheus.DefaultGatherer), "nil_registry_total")
}

// TestToContext_KeepsParentContext checks that the registry is put only into
// the returned context.
func TestToContext_KeepsParentContext(t *testing.T) {
	t.Parallel()

	parent := t.Context()
	reg := prommetrics.NewRegistry(prommetrics.WithRegisterer(prometheus.NewRegistry()))
	ctx := ToContext(parent, reg)

	assert.Same(t, reg, FromContext(ctx))
	assert.NotSame(t, reg, FromContext(parent))
}

// TestNewMetrics checks that every metric constructor takes the registry from
// the context, gives the priority to the namespace of the options and
// registers the metric in the registerer of the context.
func TestNewMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// family is the name of the metric family without a namespace.
		family string
		// create creates the metric with the given namespace in its options.
		create func(ctx context.Context, namespace string) prometheus.Collector
		// labels are the expected labels of the metric.
		labels []string
	}{
		{
			name:   "counter",
			family: "requests_total",
			create: func(ctx context.Context, namespace string) prometheus.Collector {
				return NewCounter(ctx, prometheus.CounterOpts{
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
			create: func(ctx context.Context, namespace string) prometheus.Collector {
				vec := NewCounterVec(ctx, prometheus.CounterOpts{
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
			create: func(ctx context.Context, namespace string) prometheus.Collector {
				return NewGauge(ctx, prometheus.GaugeOpts{
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
			create: func(ctx context.Context, namespace string) prometheus.Collector {
				vec := NewGaugeVec(ctx, prometheus.GaugeOpts{
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
			create: func(ctx context.Context, namespace string) prometheus.Collector {
				return NewSummary(ctx, prometheus.SummaryOpts{
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
			create: func(ctx context.Context, namespace string) prometheus.Collector {
				vec := NewSummaryVec(ctx, prometheus.SummaryOpts{
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
			create: func(ctx context.Context, namespace string) prometheus.Collector {
				return NewHistogram(ctx, prometheus.HistogramOpts{
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
			create: func(ctx context.Context, namespace string) prometheus.Collector {
				vec := NewHistogramVec(ctx, prometheus.HistogramOpts{
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
		// registryNamespace is the namespace of the registry of the context.
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
					opts := []prommetrics.Option{prommetrics.WithRegisterer(reg)}
					if sc.registryNamespace != "" {
						opts = append(opts, prommetrics.WithNamespace(sc.registryNamespace))
					}

					ctx := ToContext(t.Context(), prommetrics.NewRegistry(opts...))
					tt.create(ctx, sc.optsNamespace)

					families := testutil.GatherFamilies(t, reg)
					want := sc.wantPrefix + tt.family
					require.Contains(t, families, want)
					assert.ElementsMatch(t, tt.labels, families[want])

					// The metric is registered in the registerer of the context only.
					assert.NotContains(t, testutil.GatherFamilies(t, prometheus.DefaultGatherer), want)
				})
			}
		})
	}
}
