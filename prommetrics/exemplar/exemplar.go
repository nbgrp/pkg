// SPDX-License-Identifier: BSD-3-Clause

package exemplar

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

// FromContext makes a label to use as exemplar using the TraceID from SpanContext.
func FromContext(ctx context.Context) prometheus.Labels {
	sctx := trace.SpanFromContext(ctx).SpanContext()
	if sctx.IsSampled() {
		return prometheus.Labels{"traceID": sctx.TraceID().String()}
	}
	return nil
}
