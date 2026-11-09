# prommetrics

The creation of the Prometheus metrics of the application and the registration
of the collectors that describe the Go runtime and the process.

The module is split into two packages:

- [`github.com/nbgrp/pkg/prommetrics`](./prommetrics.go) — the registry that
  creates the metrics of the application with the common namespace, and the
  registration of the Go runtime and the process collectors.
- [`github.com/nbgrp/pkg/prommetrics/ctxprom`](./ctxprom/ctxprom.go) — the
  creation of the metrics that take the registry from the
  [context](https://pkg.go.dev/context).

The related module:

- [`github.com/nbgrp/pkg/prommetrics/exemplar`](./exemplar) — the exemplars
  that link the metrics to the OpenTelemetry traces.

## Public API

### The `prommetrics` package

```go
type Registry struct {
	// ...
}

func NewRegistry(opts ...Option) *Registry

func WithRegistry(registry *prometheus.Registry) Option
func WithRegisterer(r prometheus.Registerer) Option
func WithGatherer(g prometheus.Gatherer) Option
func WithNamespace(namespace string) Option

func (r *Registry) NewCounter(opts prometheus.CounterOpts) prometheus.Counter
func (r *Registry) NewCounterVec(opts prometheus.CounterOpts, labelValues []string) *prometheus.CounterVec
func (r *Registry) NewGauge(opts prometheus.GaugeOpts) prometheus.Gauge
func (r *Registry) NewGaugeVec(opts prometheus.GaugeOpts, labelValues []string) *prometheus.GaugeVec
func (r *Registry) NewSummary(opts prometheus.SummaryOpts) prometheus.Summary
func (r *Registry) NewSummaryVec(opts prometheus.SummaryOpts, labelValues []string) *prometheus.SummaryVec
func (r *Registry) NewHistogram(opts prometheus.HistogramOpts) prometheus.Histogram
func (r *Registry) NewHistogramVec(opts prometheus.HistogramOpts, labelValues []string) *prometheus.HistogramVec

func (r *Registry) Register(collector prometheus.Collector) error
func (r *Registry) MustRegister(collector ...prometheus.Collector)
func (r *Registry) Unregister(collector prometheus.Collector) bool
func (r *Registry) Gather() ([]*dto.MetricFamily, error)
```

`Registry` is a wrapper over the `prometheus.Registerer` and the
`prometheus.Gatherer` that keeps the namespace of the application:

- `NewRegistry` creates the registry. Without the options it uses the default
  registerer and the default gatherer of the `prometheus` package, so the
  metrics of the application are registered in the default registry.
- `WithRegistry` sets the registerer and the gatherer of the registry to the
  given `prometheus.Registry`: the metrics are registered in it and gathered
  from it.
- `WithRegisterer` sets only the registerer of the registry: the metrics are
  registered in the given registerer, but they are still gathered from the
  default gatherer.
- `WithGatherer` sets only the gatherer of the registry: the metrics are
  gathered from the given gatherer, but they are still registered in the
  default registerer.
- `WithNamespace` sets the namespace of the registry. It is applied to every
  metric whose options leave the namespace empty: the namespace of the options
  takes priority over it.
- The `New*` methods are the [`promauto`](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/promauto)
  constructors: they create the metric and register it in the registerer of the
  registry, and they panic if the registration fails.
- `Register`, `MustRegister` and `Unregister` delegate the registration of the
  collectors to the registerer of the registry. The registry implements the
  `prometheus.Registerer` interface, so it can be passed to the functions that
  expect it, e.g. to `MustRegisterGoCollectors`.
- `Gather` delegates the gathering of the metrics to the gatherer of the
  registry. The registry implements the `prometheus.Gatherer` interface, so it
  can be passed to the functions that expect it, e.g. to
  [`promhttp.HandlerFor`](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/promhttp#HandlerFor).

The registration of the Go runtime and the process collectors is described
below:

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
func ToContext(ctx context.Context, r *prommetrics.Registry) context.Context
func FromContext(ctx context.Context) *prommetrics.Registry

func NewCounter(ctx context.Context, opts prometheus.CounterOpts) prometheus.Counter
func NewCounterVec(ctx context.Context, opts prometheus.CounterOpts, labelValues []string) *prometheus.CounterVec
func NewGauge(ctx context.Context, opts prometheus.GaugeOpts) prometheus.Gauge
func NewGaugeVec(ctx context.Context, opts prometheus.GaugeOpts, labelValues []string) *prometheus.GaugeVec
func NewSummary(ctx context.Context, opts prometheus.SummaryOpts) prometheus.Summary
func NewSummaryVec(ctx context.Context, opts prometheus.SummaryOpts, labelValues []string) *prometheus.SummaryVec
func NewHistogram(ctx context.Context, opts prometheus.HistogramOpts) prometheus.Histogram
func NewHistogramVec(ctx context.Context, opts prometheus.HistogramOpts, labelValues []string) *prometheus.HistogramVec
```

The package keeps the registry of the application in the context, so the code
that creates the metrics does not have to pass it around:

- `ToContext` puts the registry into the context, `FromContext` takes it from
  there. If the context has no registry, the registry with the
  `prometheus.DefaultRegisterer` is used.
- The namespace is not stored in the context: it is set once in the registry of
  the context (`prommetrics.WithNamespace`) and is applied to every metric
  created through it. The namespace of the options takes priority over it: it
  is applied only when the options leave it empty.
- The `New*` functions create the metric and register it in the registerer of
  the registry of the context. They are the
  [`promauto`](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/promauto)
  constructors, so they panic if the registration fails.
- The Go runtime and the process collectors are registered by the
  [`prommetrics.MustRegisterGoCollectors`](#the-prommetrics-package) function in
  the registerer of the registry of the context:
  `prommetrics.MustRegisterGoCollectors(ctxprom.FromContext(ctx))`.

## Usage

```go
registry := prometheus.NewRegistry()

ctx := ctxprom.ToContext(ctx, prommetrics.NewRegistry(
	prommetrics.WithRegistry(registry),
	prommetrics.WithNamespace("myapp"),
))

prommetrics.MustRegisterGoCollectors(ctxprom.FromContext(ctx))

requests := ctxprom.NewCounterVec(ctx, prometheus.CounterOpts{
    Name: "requests_total",
    Help: "Total number of requests.",
}, []string{"code"})

requests.WithLabelValues("200").Inc()
```

The counter is registered as `myapp_requests_total`, in the registry of the
context.

The registry is a gatherer, so it can expose the metrics of the application:

```go
http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
	EnableOpenMetrics: true,
}))
```

The `EnableOpenMetrics` option turns on the OpenMetrics format, which is the
format that supports the exemplars.

The [`example`](./example) application shows the whole flow: the registry, the
context-aware metrics and the exemplars.

## Errors

`MustRegisterGoCollectors` panics if the collectors cannot be registered. The
default registerer already contains them, since the `prometheus` package
registers them on its own, so it must not be passed to the function directly:
the collectors are registered in the registry of the application, the same
registry that the metrics are created in.

```go
registry := prometheus.NewRegistry()
prommetrics.MustRegisterGoCollectors(registry) // ok

ctx := ctxprom.ToContext(ctx, prommetrics.NewRegistry(
	prommetrics.WithRegistry(registry),
))
```

The registry without an explicit registerer uses the default one, so the
collectors cannot be registered in it:

```go
prommetrics.MustRegisterGoCollectors(prommetrics.NewRegistry()) // panics
```

With `ctxprom` the collectors are registered in the registry of the context,
so the call is `prommetrics.MustRegisterGoCollectors(ctxprom.FromContext(ctx))`.
The registry of a context without one is the one with the
`prometheus.DefaultRegisterer`, so the registry must be put into the context
before the call.

The metric constructors panic if the metric cannot be registered, e.g. when a
metric with the same name is already registered.

## Concurrency

The registry and the context are read-only: the context is never modified after
it is created, and the metrics are safe for concurrent use, as any Prometheus
metric is.

`MustRegisterGoCollectors` is not safe for concurrent use with the same
registry: the collectors are registered with `MustRegister`, so a second call
with the same registry panics.
