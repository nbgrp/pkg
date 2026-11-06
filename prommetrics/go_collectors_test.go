// SPDX-License-Identifier: BSD-3-Clause

package prommetrics_test

import (
	"testing"

	. "github.com/nbgrp/pkg/prommetrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gatherFamilies returns the names of the metric families registered in the
// given registry.
func gatherFamilies(t *testing.T, reg *prometheus.Registry) map[string]bool {
	t.Helper()

	mfs, err := reg.Gather()
	require.NoError(t, err)

	families := make(map[string]bool, len(mfs))
	for _, mf := range mfs {
		families[mf.GetName()] = true
	}

	return families
}

// TestMustRegisterGoCollectors_WithMemStats checks that the legacy MemStats
// metrics are registered only when the WithMemStats option is set: without it
// they are disabled in favor of the runtime metrics.
func TestMustRegisterGoCollectors_WithMemStats(t *testing.T) {
	t.Parallel()

	plainReg := prometheus.NewRegistry()
	MustRegisterGoCollectors(plainReg)

	memStatsReg := prometheus.NewRegistry()
	MustRegisterGoCollectors(memStatsReg, WithMemStats())

	plain := gatherFamilies(t, plainReg)
	memStats := gatherFamilies(t, memStatsReg)

	assert.False(t, plain["go_memstats_alloc_bytes"])
	assert.True(t, memStats["go_memstats_alloc_bytes"])
	assert.NotEqual(t, plain, memStats)
}
