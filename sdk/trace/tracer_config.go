// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace // import "go.opentelemetry.io/otel/sdk/trace"

import "go.opentelemetry.io/otel/sdk/instrumentation"

// TracerConfigurator computes the TracerConfig for a Tracer.
//
// It is called when a Tracer is first created. It should return quickly.
type TracerConfigurator func(scope instrumentation.Scope) TracerConfig

// TracerConfig defines configurable aspects of a Tracer's behavior.
type TracerConfig struct {
	enabled bool
}

// Enabled reports whether the Tracer is enabled.
//
// Tracers are enabled by default.
func (c TracerConfig) Enabled() bool {
	return c.enabled
}

// WithTracerEnabled returns a TracerConfig with the enabled flag set.
func WithTracerEnabled(enabled bool) TracerConfig {
	return TracerConfig{enabled: enabled}
}

// WithTracerConfigurator returns a TracerProviderOption that configures the
// TracerConfigurator used when creating Tracers.
func WithTracerConfigurator(configurator TracerConfigurator) TracerProviderOption {
	return traceProviderOptionFunc(func(cfg tracerProviderConfig) tracerProviderConfig {
		cfg.tracerConfigurator = configurator
		return cfg
	})
}
