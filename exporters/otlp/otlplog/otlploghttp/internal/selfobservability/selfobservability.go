// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package selfobservability provides self-observability metrics for OTLP log exporters.
// This is an experimental feature controlled by the x.SelfObservability feature flag.
package selfobservability // import "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp/internal/selfobservability"

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/semconv/v1.37.0/otelconv"
)

var attrsPool = sync.Pool{
	New: func() any {
		// "component.name" + "component.type" + "error.type" + "server.address" + "server.port" + "http.response.status_code"
		size := 1 + 1 + 1 + 1 + 1 + 1
		s := make([]attribute.KeyValue, 0, size)
		return &s
	},
}

type ExporterMetrics struct {
	inflightMetric  otelconv.SDKExporterLogInflight
	exporterMetric  otelconv.SDKExporterLogExported
	operationMetric otelconv.SDKExporterOperationDuration
	presetAttrs     []attribute.KeyValue
}

func NewExporterMetrics(
	componentName string,
	componentType otelconv.ComponentTypeAttr,
	target string,
) (*ExporterMetrics, error) {
	em := &ExporterMetrics{}

	meter := otel.GetMeterProvider()
	m := meter.Meter(
		"go.opentelemetry.io/otel/exporters/stdout/stdouttrace",
		metric.WithInstrumentationVersion(sdk.Version()),
		metric.WithSchemaURL(semconv.SchemaURL),
	)

	var err, e error
	if em.inflightMetric, e = otelconv.NewSDKExporterLogInflight(m); e != nil {
		e = fmt.Errorf("failed to create inflight exporter metric: %w", e)
		err = errors.Join(err, e)
		otel.Handle(e)
	}

	if em.exporterMetric, e = otelconv.NewSDKExporterLogExported(m); e != nil {
		e = fmt.Errorf("filed to create exporter metric: %w", e)
		err = errors.Join(err, e)
		otel.Handle(e)
	}

	if em.operationMetric, e = otelconv.NewSDKExporterOperationDuration(m); e != nil {
		e = fmt.Errorf("failed to create exporter operation metric: %w", e)
		err = errors.Join(err, e)
		otel.Handle(e)
	}

	if err != nil {
		return nil, err
	}

	em.presetAttrs = []attribute.KeyValue{
		semconv.OTelComponentName(componentName),
		semconv.OTelComponentTypeKey.String(string(componentType)),
		semconv.ServerAddress(target),
	}
	return em, nil
}

// TrackExport tracks a export operation and records metrics.
func (em *ExporterMetrics) TrackExport(ctx context.Context, counter int64) func(err error, code int, success int64) {
	attrs := attrsPool.Get().(*[]attribute.KeyValue)
	*attrs = append(*attrs, em.presetAttrs...)

	start := time.Now()
	em.inflightMetric.Add(ctx, counter, *attrs...)

	return func(err error, code int, success int64) {
		defer func() {
			*attrs = (*attrs)[:0]
			attrsPool.Put(attrs)
		}()

		duration := time.Since(start).Seconds()
		em.inflightMetric.Add(ctx, -counter, *attrs...)
		em.exporterMetric.Add(ctx, success, *attrs...)
		if err != nil {
			em.exporterMetric.Add(ctx, counter-success, *attrs...)
			*attrs = append(*attrs, semconv.ErrorType(err))
		}
		*attrs = append(*attrs, em.operationMetric.AttrHTTPResponseStatusCode(code))
		em.operationMetric.Record(ctx, duration, *attrs...)
	}
}
