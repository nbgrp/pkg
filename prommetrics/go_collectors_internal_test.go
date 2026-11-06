// SPDX-License-Identifier: BSD-3-Clause

package prommetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		opts     []GoCollectorsOption
		expected uint
	}{
		{
			name:     "default options",
			opts:     nil,
			expected: 0,
		},
		{
			name:     "without process",
			opts:     []GoCollectorsOption{WithoutProcess()},
			expected: withoutProcess,
		},
		{
			name:     "without cpu",
			opts:     []GoCollectorsOption{WithoutCPU()},
			expected: withoutCPU,
		},
		{
			name:     "without gc",
			opts:     []GoCollectorsOption{WithoutGC()},
			expected: withoutGC,
		},
		{
			name:     "without memory",
			opts:     []GoCollectorsOption{WithoutMemory()},
			expected: withoutMemory,
		},
		{
			name:     "without scheduler",
			opts:     []GoCollectorsOption{WithoutScheduler()},
			expected: withoutScheduler,
		},
		{
			name:     "without sync",
			opts:     []GoCollectorsOption{WithoutSync()},
			expected: withoutSync,
		},
		{
			name:     "multiple options",
			opts:     []GoCollectorsOption{WithoutProcess(), WithoutCPU(), WithoutGC()},
			expected: withoutProcess | withoutCPU | withoutGC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var wo flags
			for _, opt := range tt.opts {
				opt(&wo)
			}

			assert.Equal(t, tt.expected, uint(wo))
		})
	}
}

func TestWithoutNoFlag(t *testing.T) {
	t.Parallel()

	t.Run("without single flag", func(t *testing.T) {
		t.Parallel()

		var wo flags
		WithoutGC()(&wo)

		assert.True(t, wo.noFlag(withoutProcess))
		assert.True(t, wo.noFlag(withoutCPU))
		assert.False(t, wo.noFlag(withoutGC))
		assert.True(t, wo.noFlag(withoutMemory))
		assert.True(t, wo.noFlag(withoutScheduler))
		assert.True(t, wo.noFlag(withoutSync))
	})

	t.Run("without multiple flag", func(t *testing.T) {
		t.Parallel()

		var wo flags
		WithoutGC()(&wo)
		WithoutSync()(&wo)

		assert.True(t, wo.noFlag(withoutProcess))
		assert.True(t, wo.noFlag(withoutCPU))
		assert.False(t, wo.noFlag(withoutGC))
		assert.True(t, wo.noFlag(withoutMemory))
		assert.True(t, wo.noFlag(withoutScheduler))
		assert.False(t, wo.noFlag(withoutSync))
	})
}
