// SPDX-License-Identifier: BSD-3-Clause

package prommetrics_test

import (
	"testing"

	. "github.com/nbgrp/pkg/prommetrics"
	"github.com/nbgrp/pkg/prommetrics/testutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// TestMustRegisterGoCollectors_WithMemStats checks that the legacy MemStats
// metrics are registered only when the WithMemStats option is set: without it
// they are disabled in favor of the runtime metrics.
func TestMustRegisterGoCollectors_WithMemStats(t *testing.T) {
	t.Parallel()

	plainReg := prometheus.NewRegistry()
	MustRegisterGoCollectors(plainReg)

	memStatsReg := prometheus.NewRegistry()
	MustRegisterGoCollectors(memStatsReg, WithMemStats())

	plain := testutil.GatherFamilies(t, plainReg)
	memStats := testutil.GatherFamilies(t, memStatsReg)

	assert.NotContains(t, plain, "go_memstats_alloc_bytes")
	assert.Contains(t, memStats, "go_memstats_alloc_bytes")
	assert.NotEqual(t, plain, memStats)
}
