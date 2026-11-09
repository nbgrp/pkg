# exemplar

The Prometheus exemplars built from the OpenTelemetry trace context.

An exemplar links a metric sample to the trace it belongs to, so the trace of
the request that caused the metric can be found from the metrics themselves.

## Public API

```go
func FromContext(ctx context.Context) prometheus.Labels
```

`FromContext` returns the labels of the exemplar for the span of the context:
the `traceID` label with the ID of the trace of the span.

It returns `nil` if the context has no span, or if the span is not sampled: an
unsampled trace has no ID to link to.

## Usage

The labels are passed to the metric that supports the exemplars:

```go
import (
	"github.com/nbgrp/pkg/prommetrics/exemplar"
	"github.com/prometheus/client_golang/prometheus"
)

func handle(ctx context.Context, requests *prometheus.CounterVec) {
	adder, ok := requests.With(prometheus.Labels{"code": "200"}).(prometheus.ExemplarAdder)
	if !ok {
		panic("the counter does not support exemplars")
	}

	adder.AddWithExemplar(1, exemplar.FromContext(ctx))
}
```

The exemplars are exposed in the OpenMetrics format only, so the metrics
endpoint must be requested with the `Accept: application/openmetrics-text`
header.

## Concurrency

`FromContext` only reads the context, so it is safe for concurrent use.
