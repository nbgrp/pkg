// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/nbgrp/pkg/prommetrics"
	"github.com/nbgrp/pkg/prommetrics/ctxprom"
	"github.com/nbgrp/pkg/prommetrics/exemplar"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

const (
	label1 = "label1"
	label2 = "label2"
	label3 = "label3"

	// readHeaderTimeout is the timeout of the reading of the request headers.
	readHeaderTimeout = 5 * time.Second

	// The values of the metrics with the exemplars attached.
	counterValue   = 8
	histogramValue = 0.425
)

func main() {
	registry := prommetrics.NewRegistry(prommetrics.WithRegistry(prometheus.NewRegistry()))
	ctx := ctxprom.ToContext(context.Background(), registry)

	var (
		test1 = ctxprom.NewCounterVec(ctx, prometheus.CounterOpts{
			Name: "test_total",
			Help: "Test1 help",
		}, []string{
			label1,
			label2,
		})
		test2 = ctxprom.NewHistogramVec(ctx, prometheus.HistogramOpts{
			Name:    "test_seconds",
			Help:    "Test2 help",
			Buckets: prometheus.DefBuckets,
		}, []string{
			label3,
		})
	)

	http.Handle("/metrics", promhttp.InstrumentMetricHandler(registry, promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})))

	srv := &http.Server{
		Addr:              ":8089",
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM)
	defer stop()

	tp := tracesdk.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	ctx, span := otel.Tracer("").Start(ctx, "main")
	defer span.End()

	// IMPORTANT!
	// To see exemplars in the metrics, you need to add
	// the openmetrics Accept header into the request:
	//    Accept: application/openmetrics-text

	exemplarLabel := exemplar.FromContext(ctx)

	test1.With(prometheus.Labels{label1: "A1", label2: "B1"}).Inc()
	test1.With(prometheus.Labels{label1: "A2", label2: "B2"}).Inc()
	test1.With(prometheus.Labels{label1: "A3", label2: "B3"}).Inc()

	addWithExemplar(test1.With(prometheus.Labels{label1: "A4", label2: "B4"}), counterValue, exemplarLabel)
	observeWithExemplar(test2.With(prometheus.Labels{label3: "N"}), histogramValue, exemplarLabel)

	select {
	case <-ctx.Done():
	case err := <-errCh:
		log.Printf("the metrics server: %v", err)
	}

	err := tp.Shutdown(context.WithoutCancel(ctx))
	if err != nil {
		panic(err)
	}
}

// addWithExemplar adds the value to the counter with the exemplar attached.
func addWithExemplar(c prometheus.Counter, value float64, labels prometheus.Labels) {
	adder, ok := c.(prometheus.ExemplarAdder)
	if !ok {
		panic("the counter does not support exemplars")
	}
	adder.AddWithExemplar(value, labels)
}

// observeWithExemplar observes the value in the histogram with the exemplar
// attached.
func observeWithExemplar(o prometheus.Observer, value float64, labels prometheus.Labels) {
	observer, ok := o.(prometheus.ExemplarObserver)
	if !ok {
		panic("the histogram does not support exemplars")
	}
	observer.ObserveWithExemplar(value, labels)
}
