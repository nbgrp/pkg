// SPDX-License-Identifier: BSD-3-Clause

package prommetrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Option func(reg *Registry)

func WithRegisterer(r prometheus.Registerer) Option {
	return func(reg *Registry) {
		reg.r = r
	}
}

func WithNamespace(namespace string) Option {
	return func(registerer *Registry) {
		registerer.namespace = namespace
	}
}
