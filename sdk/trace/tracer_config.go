// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace // import "go.opentelemetry.io/otel/sdk/trace"

import (
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/trace"
)

// TracerConfigurator computes the [trace.TracerConfig] for a Tracer.
//
// It is called when a Tracer is first created. It should return quickly.
type TracerConfigurator func(scope instrumentation.Scope) trace.TracerConfig

// WithTracerConfigurator returns a TracerProviderOption that configures the
// TracerConfigurator used when creating Tracers.
func WithTracerConfigurator(configurator TracerConfigurator) TracerProviderOption {
	return traceProviderOptionFunc(func(cfg tracerProviderConfig) tracerProviderConfig {
		cfg.tracerConfigurator = configurator
		return cfg
	})
}
