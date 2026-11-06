// SPDX-License-Identifier: BSD-3-Clause

package ctxprom_test

import (
	"context"
	"testing"

	"github.com/nbgrp/pkg/prommetrics"
	. "github.com/nbgrp/pkg/prommetrics/ctxprom"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gatherFamilies collects the metric families of the given gatherer and returns
// them as a map of a family name to the "name=value" pairs of its labels.
func gatherFamilies(t *testing.T, g prometheus.Gatherer) map[string][]string {
	t.Helper()

	mfs, err := g.Gather()
	require.NoError(t, err)

	families := make(map[string][]string, len(mfs))
	for _, mf := range mfs {
		var labels []string
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				labels = append(labels, l.GetName()+"="+l.GetValue())
			}
		}
		families[mf.GetName()] = labels
	}

	return families
}

func TestToContext(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	ctx := ToContext(t.Context(), reg)

	assert.Same(t, reg, FromContext(ctx))
}

func TestFromContext_Default(t *testing.T) {
	t.Parallel()

	assert.Same(t, prometheus.DefaultRegisterer, FromContext(t.Context()))
}

func TestFromContext_NilRegisterer(t *testing.T) {
	t.Parallel()

	ctx := ToContext(t.Context(), nil)

	assert.Same(t, prometheus.DefaultRegisterer, FromContext(ctx))
}

func TestToContext_KeepsParentContext(t *testing.T) {
	t.Parallel()

	parent := t.Context()
	ctx := ToContext(parent, prometheus.NewRegistry())

	assert.Same(t, prometheus.DefaultRegisterer, FromContext(parent))
	assert.NotSame(t, prometheus.DefaultRegisterer, FromContext(ctx))
}

func TestWithNamespace(t *testing.T) {
	t.Parallel()

	ctx := WithNamespace(t.Context(), "app")

	// The namespace is only visible through the metrics created with the context.
	reg := prometheus.NewRegistry()
	NewCounterFor(ToContext(ctx, reg), prometheus.CounterOpts{Name: "requests_total", Help: "Total requests."})

	assert.Contains(t, gatherFamilies(t, reg), "app_requests_total")
}

func TestWithNamespace_Overridden(t *testing.T) {
	t.Parallel()

	ctx := WithNamespace(WithNamespace(t.Context(), "app"), "worker")

	reg := prometheus.NewRegistry()
	NewCounterFor(ToContext(ctx, reg), prometheus.CounterOpts{Name: "requests_total", Help: "Total requests."})

	assert.Contains(t, gatherFamilies(t, reg), "worker_requests_total")
}

// TestNewMetrics checks that every metric constructor takes the namespace from
// the context, gives the priority to the namespace of the options and registers
// the metric in the registerer of the context.
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
				return NewCounterFor(ctx, prometheus.CounterOpts{
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
				vec := NewCounterVecFor(ctx, prometheus.CounterOpts{
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
				return NewGaugeFor(ctx, prometheus.GaugeOpts{
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
				vec := NewGaugeVecFor(ctx, prometheus.GaugeOpts{
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
				return NewSummaryFor(ctx, prometheus.SummaryOpts{
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
				vec := NewSummaryVecFor(ctx, prometheus.SummaryOpts{
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
				return NewHistogramFor(ctx, prometheus.HistogramOpts{
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
				vec := NewHistogramVecFor(ctx, prometheus.HistogramOpts{
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
		// ctxNamespace is the namespace stored in the context.
		ctxNamespace string
		// optsNamespace is the namespace set in the options of the metric.
		optsNamespace string
		// wantPrefix is the expected prefix of the name of the metric family.
		wantPrefix string
	}{
		{
			name:         "namespace from context",
			ctxNamespace: "app",
			wantPrefix:   "app_",
		},
		{
			name:          "namespace from options",
			ctxNamespace:  "app",
			optsNamespace: "worker",
			wantPrefix:    "worker_",
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
					ctx := ToContext(t.Context(), reg)
					if sc.ctxNamespace != "" {
						ctx = WithNamespace(ctx, sc.ctxNamespace)
					}

					tt.create(ctx, sc.optsNamespace)

					families := gatherFamilies(t, reg)
					want := sc.wantPrefix + tt.family
					require.Contains(t, families, want)
					assert.ElementsMatch(t, tt.labels, families[want])

					// The metric is registered in the registerer of the context only.
					assert.NotContains(t, gatherFamilies(t, prometheus.DefaultGatherer), want)
				})
			}
		})
	}
}

func TestMustRegisterGoCollectors(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	MustRegisterGoCollectors(ToContext(t.Context(), reg))

	families := gatherFamilies(t, reg)
	assert.Contains(t, families, "go_goroutines")
	assert.Contains(t, families, "process_cpu_seconds_total")
}

func TestMustRegisterGoCollectors_Options(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	MustRegisterGoCollectors(ToContext(t.Context(), reg), prommetrics.WithoutProcess())

	families := gatherFamilies(t, reg)
	assert.Contains(t, families, "go_goroutines")
	assert.NotContains(t, families, "process_cpu_seconds_total")
}

// TestMustRegisterGoCollectors_DefaultRegisterer checks that the collectors are
// registered in the default registerer, which already contains them since the
// prometheus package registers them on its own.
func TestMustRegisterGoCollectors_DefaultRegisterer(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		MustRegisterGoCollectors(t.Context())
	})
}
