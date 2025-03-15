// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package global // import "go.opentelemetry.io/otel/internal/global"

import (
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"unique"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type (
	errorHandlerHolder struct {
		eh ErrorHandler
	}

	tracerProviderHolder struct {
		flag     bool
		tp       unique.Handle[trace.TracerProvider]
		nonCmpTp trace.TracerProvider
	}

	propagatorsHolder struct {
		tm propagation.TextMapPropagator
	}

	meterProviderHolder struct {
		mp metric.MeterProvider
	}
)

var (
	globalErrorHandler  = defaultErrorHandler()
	globalTracer        = defaultTracerValue()
	globalPropagators   = defaultPropagatorsValue()
	globalMeterProvider = defaultMeterProvider()

	delegateErrorHandlerOnce      sync.Once
	delegateTraceOnce             sync.Once
	delegateTextMapPropagatorOnce sync.Once
	delegateMeterOnce             sync.Once
)

// GetErrorHandler returns the global ErrorHandler instance.
//
// The default ErrorHandler instance returned will log all errors to STDERR
// until an override ErrorHandler is set with SetErrorHandler. All
// ErrorHandler returned prior to this will automatically forward errors to
// the set instance instead of logging.
//
// Subsequent calls to SetErrorHandler after the first will not forward errors
// to the new ErrorHandler for prior returned instances.
func GetErrorHandler() ErrorHandler {
	return globalErrorHandler.Load().(errorHandlerHolder).eh
}

// SetErrorHandler sets the global ErrorHandler to h.
//
// The first time this is called all ErrorHandler previously returned from
// GetErrorHandler will send errors to h instead of the default logging
// ErrorHandler. Subsequent calls will set the global ErrorHandler, but not
// delegate errors to h.
func SetErrorHandler(h ErrorHandler) {
	current := GetErrorHandler()

	if _, cOk := current.(*ErrDelegator); cOk {
		if _, ehOk := h.(*ErrDelegator); ehOk && current == h {
			// Do not assign to the delegate of the default ErrDelegator to be
			// itself.
			Error(
				errors.New("no ErrorHandler delegate configured"),
				"ErrorHandler remains its current value.",
			)
			return
		}
	}

	delegateErrorHandlerOnce.Do(func() {
		if def, ok := current.(*ErrDelegator); ok {
			def.setDelegate(h)
		}
	})
	globalErrorHandler.Store(errorHandlerHolder{eh: h})
}

func GetTracerProvider() (trace.TracerProvider, bool) {
	flag := globalTracer.Load().(tracerProviderHolder).flag
	if !flag {
		return globalTracer.Load().(tracerProviderHolder).nonCmpTp, false
	}

	tp := globalTracer.Load().(tracerProviderHolder).tp
	return tp.Value(), true
}

// TracerProvider is the internal implementation for global.TracerProvider.
func TracerProvider() trace.TracerProvider {
	tp, _ := GetTracerProvider()
	return tp
}

func isCmp(t any) bool {
	ty := reflect.TypeOf(t)
	return ty.Comparable()
}

func getTraceProviderHandle() unique.Handle[trace.TracerProvider] {
	return globalTracer.Load().(tracerProviderHolder).tp
}

// SetTracerProvider is the internal implementation for global.SetTracerProvider.
func SetTracerProvider(tp trace.TracerProvider) {
	current := TracerProvider()

	if isCmp(tp) {
		hd := unique.Make[trace.TracerProvider](tp)
		hd1 := getTraceProviderHandle()

		if hd == hd1 {
			Error(
				errors.New("no delegate configured in tracer provider"),
				"Setting tracer provider to its current value. No delegate will be configured",
			)
			return
		}

		globalTracer.Store(
			tracerProviderHolder{
				tp:   unique.Make[trace.TracerProvider](tp),
				flag: true,
			})
	} else {
		globalTracer.Store(
			tracerProviderHolder{
				nonCmpTp: tp,
				flag:     false,
			})
	}

	delegateTraceOnce.Do(func() {
		if def, ok := current.(*tracerProvider); ok {
			def.setDelegate(tp)
		}
	})
}

// TextMapPropagator is the internal implementation for global.TextMapPropagator.
func TextMapPropagator() propagation.TextMapPropagator {
	return globalPropagators.Load().(propagatorsHolder).tm
}

// SetTextMapPropagator is the internal implementation for global.SetTextMapPropagator.
func SetTextMapPropagator(p propagation.TextMapPropagator) {
	current := TextMapPropagator()

	if _, cOk := current.(*textMapPropagator); cOk {
		if _, pOk := p.(*textMapPropagator); pOk && current == p {
			// Do not assign the default delegating TextMapPropagator to
			// delegate to itself.
			Error(
				errors.New("no delegate configured in text map propagator"),
				"Setting text map propagator to its current value. No delegate will be configured",
			)
			return
		}
	}

	// For the textMapPropagator already returned by TextMapPropagator
	// delegate to p.
	delegateTextMapPropagatorOnce.Do(func() {
		if def, ok := current.(*textMapPropagator); ok {
			def.SetDelegate(p)
		}
	})
	// Return p when subsequent calls to TextMapPropagator are made.
	globalPropagators.Store(propagatorsHolder{tm: p})
}

// MeterProvider is the internal implementation for global.MeterProvider.
func MeterProvider() metric.MeterProvider {
	return globalMeterProvider.Load().(meterProviderHolder).mp
}

// SetMeterProvider is the internal implementation for global.SetMeterProvider.
func SetMeterProvider(mp metric.MeterProvider) {
	current := MeterProvider()
	if _, cOk := current.(*meterProvider); cOk {
		if _, mpOk := mp.(*meterProvider); mpOk && current == mp {
			// Do not assign the default delegating MeterProvider to delegate
			// to itself.
			Error(
				errors.New("no delegate configured in meter provider"),
				"Setting meter provider to its current value. No delegate will be configured",
			)
			return
		}
	}

	delegateMeterOnce.Do(func() {
		if def, ok := current.(*meterProvider); ok {
			def.setDelegate(mp)
		}
	})
	globalMeterProvider.Store(meterProviderHolder{mp: mp})
}

func defaultErrorHandler() *atomic.Value {
	v := &atomic.Value{}
	v.Store(errorHandlerHolder{eh: &ErrDelegator{}})
	return v
}

func defaultTracerValue() *atomic.Value {
	v := &atomic.Value{}
	v.Store(
		tracerProviderHolder{
			tp:       unique.Make[trace.TracerProvider](&tracerProvider{}),
			nonCmpTp: &tracerProvider{},
			flag:     true,
		})
	return v
}

func defaultPropagatorsValue() *atomic.Value {
	v := &atomic.Value{}
	v.Store(propagatorsHolder{tm: newTextMapPropagator()})
	return v
}

func defaultMeterProvider() *atomic.Value {
	v := &atomic.Value{}
	v.Store(meterProviderHolder{mp: &meterProvider{}})
	return v
}
