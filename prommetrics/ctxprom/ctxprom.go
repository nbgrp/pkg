// SPDX-License-Identifier: BSD-3-Clause

package ctxprom

import (
	"context"

	"github.com/nbgrp/pkg/prommetrics"
	"github.com/prometheus/client_golang/prometheus"
)

type registererKey struct{}

// ToContext puts the registry into the context.
func ToContext(ctx context.Context, r *prommetrics.Registry) context.Context {
	return context.WithValue(ctx, registererKey{}, r)
}

// FromContext returns the registry of the context. If the context has no
// registry, the registry with the default registerer is returned.
func FromContext(ctx context.Context) *prommetrics.Registry {
	if r, ok := ctx.Value(registererKey{}).(*prommetrics.Registry); ok && r != nil {
		return r
	}
	return prommetrics.NewRegistry(prommetrics.WithRegisterer(prometheus.DefaultRegisterer))
}

func NewCounter(ctx context.Context, opts prometheus.CounterOpts) prometheus.Counter {
	return FromContext(ctx).NewCounter(opts)
}

func NewCounterVec(ctx context.Context, opts prometheus.CounterOpts, labelValues []string) *prometheus.CounterVec {
	return FromContext(ctx).NewCounterVec(opts, labelValues)
}

func NewGauge(ctx context.Context, opts prometheus.GaugeOpts) prometheus.Gauge {
	return FromContext(ctx).NewGauge(opts)
}

func NewGaugeVec(ctx context.Context, opts prometheus.GaugeOpts, labelValues []string) *prometheus.GaugeVec {
	return FromContext(ctx).NewGaugeVec(opts, labelValues)
}

func NewSummary(ctx context.Context, opts prometheus.SummaryOpts) prometheus.Summary {
	return FromContext(ctx).NewSummary(opts)
}

func NewSummaryVec(ctx context.Context, opts prometheus.SummaryOpts, labelValues []string) *prometheus.SummaryVec {
	return FromContext(ctx).NewSummaryVec(opts, labelValues)
}

func NewHistogram(ctx context.Context, opts prometheus.HistogramOpts) prometheus.Histogram {
	return FromContext(ctx).NewHistogram(opts)
}

func NewHistogramVec(ctx context.Context, opts prometheus.HistogramOpts, labelValues []string) *prometheus.HistogramVec {
	return FromContext(ctx).NewHistogramVec(opts, labelValues)
}
