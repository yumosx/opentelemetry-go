package selfobservability

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
	semconv "go.opentelemetry.io/otel/semconv/v1.36.0"
	"go.opentelemetry.io/otel/semconv/v1.36.0/otelconv"
)

var attrsPool = sync.Pool{
	New: func() interface{} {
		size := 1 + 1 + 1 + 1
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

func NewExporterMetrics(componentName string, componentType otelconv.ComponentTypeAttr, port int) (*ExporterMetrics, error) {
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
	}
	return em, nil
}

func (em *ExporterMetrics) TrackExport(ctx context.Context, counter int64) func(err error, code int, success int64) {
	attrs := attrsPool.Get().(*[]attribute.KeyValue)
	*attrs = append(*attrs, em.presetAttrs...)

	start := time.Now()
	em.inflightMetric.Add(ctx, counter, *attrs...)

	return func(err error, code int, success int64) {
		defer func() {
			*attrs = (*attrs)[:]
			attrsPool.Put(attrs)
		}()

		duration := time.Now().Sub(start).Seconds()
		em.inflightMetric.Add(ctx, -counter, *attrs...)
		em.exporterMetric.Add(ctx, success, *attrs...)
		if err != nil {
			em.exporterMetric.Add(ctx, counter-success, *attrs...)
			*attrs = append(*attrs, semconv.ErrorType(err))
		}
		em.operationMetric.Record(ctx, duration, *attrs...)
	}
}
