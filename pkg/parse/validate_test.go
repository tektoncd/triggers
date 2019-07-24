package parse

import (
	"fmt"
	"testing"
)

func TestInterpolate(t *testing.T) {
	binding := "something.something"
	interpolatedBinding := Interpolate(binding)
	if interpolatedBinding != fmt.Sprintf("$(%s)",binding) {
		t.Error("Unexpected interpolated value")
	}
}

func TestEventRegex(t *testing.T) {
	validPaths := []string{
		"$(event)",
		"$(event.a-b)",
		"$(event.a1)",
		"$(event.a.b)",
		"$(event.a.b.c)",
	}
	invalidPaths := []string{
		"$event",
		"$[event]",
		"${event}",
		"$(event.1)",
		"$(event..)",
		"$(event.$a)",
		"event.a",
		"event",
		"${{event}",
		"${event",
	}
	for _, validPath := range validPaths {
		if !ValidEventBinding(validPath) {
			t.Errorf("Path '%s' does not match against event regex '%s'", validPath, eventRegex.String())
		}
	}
	for _, invalidPath := range invalidPaths {
		if ValidEventBinding(invalidPath) {
			t.Errorf("Path '%s' should not have matched against event regex '%s'", invalidPath, eventRegex.String())
		}
	}
}

func TestParamsRegex(t *testing.T) {
	validPaths := []string{
		"$(params.a)",
		"$(params.a-b)",
		"$(params.a1)",
	}
	invalidPaths := []string{
		"$(params)",
		"$(param.a)",
		"$(params.a.b)",
		"$params",
		"$[params]",
		"${params}",
		"$(params.1)",
		"$(params..)",
		"$(params.$a)",
		"params.a",
		"params",
		"${{params}",
		"${params",
	}
	for _, validPath := range validPaths {
		if !ValidTemplateBinding(validPath) {
			t.Errorf("Path '%s' does not match against params regex '%s'", validPath, paramsRegex.String())
		}
	}
	for _, invalidPath := range invalidPaths {
		if ValidTemplateBinding(invalidPath) {
			t.Errorf("Path '%s' should not have matched against params regex '%s'", invalidPath, paramsRegex.String())
		}
	}
}
