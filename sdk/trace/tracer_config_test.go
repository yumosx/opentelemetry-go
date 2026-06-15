// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/sdk/instrumentation"
	apitrace "go.opentelemetry.io/otel/trace"
)

func TestTracerConfiguratorDisablesTracer(t *testing.T) {
	p := NewTracerProvider(WithTracerConfigurator(func(scope instrumentation.Scope) TracerBehaviorConfig {
		if scope.Name == "disabled" {
			return WithTracerEnabled(false)
		}
		return TracerBehaviorConfig{}
	}))

	enabled := p.Tracer("enabled")
	disabled := p.Tracer("disabled")

	require.IsType(t, &tracer{}, enabled)
	require.IsType(t, &tracer{}, disabled)

	_, enabledSpan := enabled.Start(t.Context(), "span")
	assert.True(t, enabledSpan.IsRecording())

	_, disabledSpan := disabled.Start(t.Context(), "span")
	assert.False(t, disabledSpan.IsRecording())
}

func TestTracerConfiguratorDefaultEnablesTracer(t *testing.T) {
	p := NewTracerProvider(WithTracerConfigurator(func(instrumentation.Scope) TracerBehaviorConfig {
		return TracerBehaviorConfig{}
	}))

	tr := p.Tracer("enabled")
	require.IsType(t, &tracer{}, tr)

	_, span := tr.Start(t.Context(), "span")
	assert.True(t, span.IsRecording())
}

func TestTracerConfiguratorPreservesScope(t *testing.T) {
	p := NewTracerProvider(WithTracerConfigurator(func(scope instrumentation.Scope) TracerBehaviorConfig {
		assert.Equal(t, "my-tracer", scope.Name)
		assert.Equal(t, "1.0.0", scope.Version)
		return TracerBehaviorConfig{}
	}))

	tr := p.Tracer("my-tracer", apitrace.WithInstrumentationVersion("1.0.0"))
	require.IsType(t, &tracer{}, tr)
	assert.Equal(t, "1.0.0", tr.(*tracer).instrumentationScope.Version)
}

func TestUpdateTracerConfigurator(t *testing.T) {
	var disabled bool
	p := NewTracerProvider(WithTracerConfigurator(func(scope instrumentation.Scope) TracerBehaviorConfig {
		if scope.Name == "library" && disabled {
			return WithTracerEnabled(false)
		}
		return TracerBehaviorConfig{}
	}))

	tr := p.Tracer("library")
	require.IsType(t, &tracer{}, tr)

	_, span := tr.Start(t.Context(), "before")
	assert.True(t, span.IsRecording())

	disabled = true
	p.UpdateTracerConfigurator(func(scope instrumentation.Scope) TracerBehaviorConfig {
		if scope.Name == "library" && disabled {
			return WithTracerEnabled(false)
		}
		return TracerBehaviorConfig{}
	})

	// Same Tracer instance; behavior changes on the next Start().
	assert.Same(t, tr, p.Tracer("library"))

	_, span = tr.Start(t.Context(), "after")
	assert.False(t, span.IsRecording())

	disabled = false
	p.UpdateTracerConfigurator(func(scope instrumentation.Scope) TracerBehaviorConfig {
		if scope.Name == "library" && disabled {
			return WithTracerEnabled(false)
		}
		return TracerBehaviorConfig{}
	})

	_, span = tr.Start(t.Context(), "re-enabled")
	assert.True(t, span.IsRecording())
}
