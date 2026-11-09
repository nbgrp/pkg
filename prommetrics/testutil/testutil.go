// SPDX-License-Identifier: BSD-3-Clause

//go:build testing

package testutil

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// GatherFamilies collects the metric families of the given gatherer and returns
// them as a map of a family name to the "name=value" pairs of its labels.
func GatherFamilies(t *testing.T, g prometheus.Gatherer) map[string][]string {
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
