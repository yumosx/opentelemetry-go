// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/sdk/instrumentation"
	apitrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestTracerConfiguratorDisablesTracer(t *testing.T) {
	p := NewTracerProvider(WithTracerConfigurator(func(scope instrumentation.Scope) apitrace.TracerConfig {
		if scope.Name == "disabled" {
			return apitrace.NewTracerConfig(apitrace.WithTracerEnabled(false))
		}
		return apitrace.NewTracerConfig(apitrace.WithTracerEnabled(true))
	}))

	enabled := p.Tracer("enabled")
	disabled := p.Tracer("disabled")

	require.IsType(t, &tracer{}, enabled)
	require.IsType(t, noop.Tracer{}, disabled)

	_, enabledSpan := enabled.Start(t.Context(), "span")
	assert.True(t, enabledSpan.IsRecording())

	_, disabledSpan := disabled.Start(t.Context(), "span")
	assert.False(t, disabledSpan.IsRecording())
}


func TestTracerConfiguratorPreservesScope(t *testing.T) {
	p := NewTracerProvider(WithTracerConfigurator(func(scope instrumentation.Scope) apitrace.TracerConfig {
		assert.Equal(t, "my-tracer", scope.Name)
		assert.Equal(t, "1.0.0", scope.Version)
		return apitrace.NewTracerConfig(apitrace.WithTracerEnabled(true))
	}))

	tr := p.Tracer("my-tracer", apitrace.WithInstrumentationVersion("1.0.0"))
	require.IsType(t, &tracer{}, tr)
	assert.Equal(t, "1.0.0", tr.(*tracer).instrumentationScope.Version)
}
