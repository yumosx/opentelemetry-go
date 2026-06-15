// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace // import "go.opentelemetry.io/otel/sdk/trace"

import "go.opentelemetry.io/otel/sdk/instrumentation"

// TracerConfigurator computes SDK behavioral config for a Tracer.
//
// It is called when a Tracer is first created and when the Provider's
// configurator is updated. It should return quickly.
type TracerConfigurator func(scope instrumentation.Scope) TracerBehaviorConfig

// TracerBehaviorConfig defines configurable aspects of a Tracer's behavior.
type TracerBehaviorConfig struct {
	enabled *bool
}

// Enabled reports whether the Tracer is enabled.
//
// Tracers are enabled by default.
func (c TracerBehaviorConfig) Enabled() bool {
	if c.enabled == nil {
		return true
	}
	return *c.enabled
}

// WithTracerEnabled returns a TracerBehaviorConfig with the enabled flag set.
func WithTracerEnabled(enabled bool) TracerBehaviorConfig {
	return TracerBehaviorConfig{enabled: &enabled}
}

// WithTracerConfigurator returns a TracerProviderOption that configures the
// TracerConfigurator used when creating Tracers.
func WithTracerConfigurator(configurator TracerConfigurator) TracerProviderOption {
	return traceProviderOptionFunc(func(cfg tracerProviderConfig) tracerProviderConfig {
		cfg.tracerConfigurator = configurator
		return cfg
	})
}

// UpdateTracerConfigurator replaces the Provider's configurator and updates
// all outstanding Tracers to match the new configuration.
func (p *TracerProvider) UpdateTracerConfigurator(configurator TracerConfigurator) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.tracerConfigurator = configurator
	for _, tr := range p.namedTracer {
		tr.setEnabled(p.tracerEnabled(tr.instrumentationScope))
	}
}

func (p *TracerProvider) tracerEnabled(scope instrumentation.Scope) bool {
	if p.tracerConfigurator == nil {
		return true
	}
	return p.tracerConfigurator(scope).Enabled()
}
