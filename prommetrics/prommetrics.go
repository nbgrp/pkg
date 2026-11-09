// SPDX-License-Identifier: BSD-3-Clause

package prommetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var _ prometheus.Registerer = (*Registry)(nil)

type Registry struct {
	r         prometheus.Registerer
	namespace string
}

func NewRegistry(opts ...Option) *Registry {
	r := &Registry{}
	for _, opt := range opts {
		opt(r)
	}
	if r.r == nil {
		r.r = prometheus.NewRegistry()
	}
	return r
}

func (r *Registry) Register(collector prometheus.Collector) error {
	return r.r.Register(collector)
}

func (r *Registry) MustRegister(collector ...prometheus.Collector) {
	r.r.MustRegister(collector...)
}

func (r *Registry) Unregister(collector prometheus.Collector) bool {
	return r.r.Unregister(collector)
}

func (r *Registry) NewCounter(opts prometheus.CounterOpts) prometheus.Counter {
	if opts.Namespace == "" {
		opts.Namespace = r.namespace
	}
	return promauto.With(r.r).NewCounter(opts)
}

func (r *Registry) NewCounterVec(opts prometheus.CounterOpts, labelValues []string) *prometheus.CounterVec {
	if opts.Namespace == "" {
		opts.Namespace = r.namespace
	}
	return promauto.With(r.r).NewCounterVec(opts, labelValues)
}

func (r *Registry) NewGauge(opts prometheus.GaugeOpts) prometheus.Gauge {
	if opts.Namespace == "" {
		opts.Namespace = r.namespace
	}
	return promauto.With(r.r).NewGauge(opts)
}

func (r *Registry) NewGaugeVec(opts prometheus.GaugeOpts, labelValues []string) *prometheus.GaugeVec {
	if opts.Namespace == "" {
		opts.Namespace = r.namespace
	}
	return promauto.With(r.r).NewGaugeVec(opts, labelValues)
}

func (r *Registry) NewSummary(opts prometheus.SummaryOpts) prometheus.Summary {
	if opts.Namespace == "" {
		opts.Namespace = r.namespace
	}
	return promauto.With(r.r).NewSummary(opts)
}

func (r *Registry) NewSummaryVec(opts prometheus.SummaryOpts, labelValues []string) *prometheus.SummaryVec {
	if opts.Namespace == "" {
		opts.Namespace = r.namespace
	}
	return promauto.With(r.r).NewSummaryVec(opts, labelValues)
}

func (r *Registry) NewHistogram(opts prometheus.HistogramOpts) prometheus.Histogram {
	if opts.Namespace == "" {
		opts.Namespace = r.namespace
	}
	return promauto.With(r.r).NewHistogram(opts)
}

func (r *Registry) NewHistogramVec(opts prometheus.HistogramOpts, labelValues []string) *prometheus.HistogramVec {
	if opts.Namespace == "" {
		opts.Namespace = r.namespace
	}
	return promauto.With(r.r).NewHistogramVec(opts, labelValues)
}
