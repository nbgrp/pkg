// SPDX-License-Identifier: BSD-3-Clause

package exemplar_test

import (
	"testing"

	. "github.com/nbgrp/pkg/prommetrics/exemplar"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

var (
	traceID = trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	spanID  = trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
)

// TestFromContext_NoSpan checks that the context without a span has no
// exemplar.
func TestFromContext_NoSpan(t *testing.T) {
	t.Parallel()

	assert.Nil(t, FromContext(t.Context()))
}

// TestFromContext_NotSampled checks that the span that is not sampled has no
// exemplar, since its trace has no ID to link to.
func TestFromContext_NotSampled(t *testing.T) {
	t.Parallel()

	ctx := trace.ContextWithSpanContext(t.Context(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	}))

	assert.Nil(t, FromContext(ctx))
}

// TestFromContext_Sampled checks that the sampled span gives the exemplar
// with the ID of its trace.
func TestFromContext_Sampled(t *testing.T) {
	t.Parallel()

	ctx := trace.ContextWithSpanContext(t.Context(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	}))

	assert.Equal(t, prometheus.Labels{"traceID": traceID.String()}, FromContext(ctx))
}
