// SPDX-License-Identifier: BSD-3-Clause

package ctxprom

import (
	"context"

	"github.com/nbgrp/pkg/prommetrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type registererKey struct{}
type namespaceKey struct{}

func ToContext(ctx context.Context, r prometheus.Registerer) context.Context {
	return context.WithValue(ctx, registererKey{}, r)
}

func FromContext(ctx context.Context) prometheus.Registerer {
	if r, ok := ctx.Value(registererKey{}).(prometheus.Registerer); ok {
		return r
	}
	return prometheus.DefaultRegisterer
}

func WithNamespace(ctx context.Context, namespace string) context.Context {
	return context.WithValue(ctx, namespaceKey{}, namespace)
}

func NewCounterFor(ctx context.Context, opts prometheus.CounterOpts) prometheus.Counter {
	if opts.Namespace == "" {
		opts.Namespace, _ = ctx.Value(namespaceKey{}).(string)
	}
	return promauto.With(FromContext(ctx)).NewCounter(opts)
}

func NewCounterVecFor(ctx context.Context, opts prometheus.CounterOpts, labelValues []string) *prometheus.CounterVec {
	if opts.Namespace == "" {
		opts.Namespace, _ = ctx.Value(namespaceKey{}).(string)
	}
	return promauto.With(FromContext(ctx)).NewCounterVec(opts, labelValues)
}

func NewGaugeFor(ctx context.Context, opts prometheus.GaugeOpts) prometheus.Gauge {
	if opts.Namespace == "" {
		opts.Namespace, _ = ctx.Value(namespaceKey{}).(string)
	}
	return promauto.With(FromContext(ctx)).NewGauge(opts)
}

func NewGaugeVecFor(ctx context.Context, opts prometheus.GaugeOpts, labelValues []string) *prometheus.GaugeVec {
	if opts.Namespace == "" {
		opts.Namespace, _ = ctx.Value(namespaceKey{}).(string)
	}
	return promauto.With(FromContext(ctx)).NewGaugeVec(opts, labelValues)
}

func NewSummaryFor(ctx context.Context, opts prometheus.SummaryOpts) prometheus.Summary {
	if opts.Namespace == "" {
		opts.Namespace, _ = ctx.Value(namespaceKey{}).(string)
	}
	return promauto.With(FromContext(ctx)).NewSummary(opts)
}

func NewSummaryVecFor(ctx context.Context, opts prometheus.SummaryOpts, labelValues []string) *prometheus.SummaryVec {
	if opts.Namespace == "" {
		opts.Namespace, _ = ctx.Value(namespaceKey{}).(string)
	}
	return promauto.With(FromContext(ctx)).NewSummaryVec(opts, labelValues)
}

func NewHistogramFor(ctx context.Context, opts prometheus.HistogramOpts) prometheus.Histogram {
	if opts.Namespace == "" {
		opts.Namespace, _ = ctx.Value(namespaceKey{}).(string)
	}
	return promauto.With(FromContext(ctx)).NewHistogram(opts)
}

func NewHistogramVecFor(ctx context.Context, opts prometheus.HistogramOpts, labelValues []string) *prometheus.HistogramVec {
	if opts.Namespace == "" {
		opts.Namespace, _ = ctx.Value(namespaceKey{}).(string)
	}
	return promauto.With(FromContext(ctx)).NewHistogramVec(opts, labelValues)
}

// MustRegisterGoCollectors registers the Go runtime and the process collectors
// in the registerer of the context.
//
// It panics if the collectors cannot be registered. The default registerer
// already contains them, since the prometheus package registers them on its
// own, so the registerer must be put into the context before the call.
func MustRegisterGoCollectors(ctx context.Context, opts ...prommetrics.GoCollectorsOption) {
	prommetrics.MustRegisterGoCollectors(FromContext(ctx), opts...)
}
