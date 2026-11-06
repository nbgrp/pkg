# prommetrics

Registration of the Prometheus collectors that describe the Go runtime and the
process, and the creation of the application metrics that follow the namespace
and the registry of the [context](https://pkg.go.dev/context).

The module is split into two packages:

- [`github.com/nbgrp/pkg/prommetrics`](./go_collectors.go) — the registration of
  the Go runtime and the process collectors.
- [`github.com/nbgrp/pkg/prommetrics/ctxprom`](./ctxprom/ctxprom.go) — the
  creation of the metrics that take the registry and the namespace from the
  context.

## Public API

### The `prommetrics` package

```go
type GoCollectorsOption func(*flags)

func WithoutProcess() GoCollectorsOption
func WithoutCPU() GoCollectorsOption
func WithoutGC() GoCollectorsOption
func WithoutMemory() GoCollectorsOption
func WithoutScheduler() GoCollectorsOption
func WithoutSync() GoCollectorsOption

func WithMemStats() GoCollectorsOption

func MustRegisterGoCollectors(r prometheus.Registerer, opts ...GoCollectorsOption)
```

`MustRegisterGoCollectors` registers the Go collector and the process collector
in the given registry. The options turn the individual groups of the metrics
off:

| Option           | Metrics that are removed                                     |
|------------------|--------------------------------------------------------------|
| `WithoutProcess` | the `process_*` metrics                                      |
| `WithoutCPU`     | the `go_cpu_classes_*` metrics                               |
| `WithoutGC`      | the `go_gc_*` metrics                                        |
| `WithoutMemory`  | the `go_memory_classes_*` metrics                           |
| `WithoutScheduler` | the `go_sched_*` metrics                                  |
| `WithoutSync`    | the `go_sync_*` metrics                                      |

The runtime metrics come from the
[`runtime/metrics`](https://pkg.go.dev/runtime/metrics) package, so their exact
set depends on the Go version.

`WithMemStats` is the only option that turns the metrics **on**: it keeps the
legacy `go_memstats_*` family, which is based on `runtime.MemStats`. Without
it the family is disabled, since the runtime metrics cover the same ground.

```go
prommetrics.MustRegisterGoCollectors(registry)
// 79 metric families: the runtime and the process metrics.

prommetrics.MustRegisterGoCollectors(registry, prommetrics.WithMemStats())
// 100 metric families: the same plus the legacy go_memstats_* ones.
```

The options are independent of each other, so they can be combined freely.

### The `ctxprom` package

```go
func ToContext(ctx context.Context, r prometheus.Registerer) context.Context
func FromContext(ctx context.Context) prometheus.Registerer
func WithNamespace(ctx context.Context, namespace string) context.Context

func NewCounterFor(ctx context.Context, opts prometheus.CounterOpts) prometheus.Counter
func NewCounterVecFor(ctx context.Context, opts prometheus.CounterOpts, labelValues []string) *prometheus.CounterVec
func NewGaugeFor(ctx context.Context, opts prometheus.GaugeOpts) prometheus.Gauge
func NewGaugeVecFor(ctx context.Context, opts prometheus.GaugeOpts, labelValues []string) *prometheus.GaugeVec
func NewSummaryFor(ctx context.Context, opts prometheus.SummaryOpts) prometheus.Summary
func NewSummaryVecFor(ctx context.Context, opts prometheus.SummaryOpts, labelValues []string) *prometheus.SummaryVec
func NewHistogramFor(ctx context.Context, opts prometheus.HistogramOpts) prometheus.Histogram
func NewHistogramVecFor(ctx context.Context, opts prometheus.HistogramOpts, labelValues []string) *prometheus.HistogramVec

func MustRegisterGoCollectors(ctx context.Context, opts ...prommetrics.GoCollectorsOption)
```

The package keeps the registry and the namespace of the application in the
context, so the code that creates the metrics does not have to pass them
around:

- `ToContext` puts the registry into the context, `FromContext` takes it from
  there. If the context has no registry, `prometheus.DefaultRegisterer` is
  used.
- `WithNamespace` puts the namespace into the context. The namespace of the
  options takes priority over it: it is applied only when the options leave it
  empty.
- The `New*For` functions create the metric and register it in the registry of
  the context. They are the [`promauto`](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/promauto)
  constructors, so they panic if the registration fails.
- `MustRegisterGoCollectors` registers the collectors of the
  [`prommetrics`](#the-prommetrics-package) package in the registry of the
  context.

## Usage

```go
ctx := ctxprom.ToContext(ctx, registry)
ctx = ctxprom.WithNamespace(ctx, "myapp")

ctxprom.MustRegisterGoCollectors(ctx)

requests := ctxprom.NewCounterVecFor(ctx, prometheus.CounterOpts{
    Name: "requests_total",
    Help: "Total number of requests.",
}, []string{"code"})

requests.WithLabelValues("200").Inc()
```

The counter is registered as `myapp_requests_total`, in the registry of the
context.

## Errors

`MustRegisterGoCollectors` panics if the collectors cannot be registered. The
default registerer already contains them, since the `prometheus` package
registers them on its own, so the registry must be put into the context before
the call:

```go
ctx := ctxprom.ToContext(ctx, prometheus.NewRegistry())

ctxprom.MustRegisterGoCollectors(ctx) // ok
```

The same applies to the `prommetrics` package: the default registerer must not
be passed to `MustRegisterGoCollectors` directly.

The metric constructors panic if the metric cannot be registered, e.g. when a
metric with the same name is already registered.

## Concurrency

The registry and the namespace of the context are read-only: the context is
never modified after it is created, and the metrics are safe for concurrent
use, as any Prometheus metric is.

`MustRegisterGoCollectors` is not safe for concurrent use with the same
registry: the collectors are registered with `MustRegister`, so a second call
with the same registry panics.
