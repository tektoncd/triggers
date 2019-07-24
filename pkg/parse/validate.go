package parse

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// InterpolationBaseMock mocks the interpolation base
	InterpolationBaseMock string = "x"
	// InterpolationSyntax represents the syntax used by triggers to specify interpolation
	InterpolationSyntax string = "$(" + InterpolationBaseMock + ")"
	// interpolationRegex is the regex formatted version of the InterpolationSyntax
	interpolationRegex string = "\\$\\(" + InterpolationBaseMock + "\\)"
	// interpolationFieldRegexString is the regex string specifying legal field names within data object
	interpolationFieldRegexString string = "[_a-zA-Z][_0-9a-zA-Z-]*"
	// EventInterpolationBase is the base string to use within the InterpolationSyntax to specify the event object
	EventInterpolationBase string = "event"
	// ParamsInterpolationBase is the base string to use within the InterpolationSyntax to specify the event object
	ParamsInterpolationBase string = "params"
)

// If template fields do not match this regex, their syntax is malformed.
// This only a preliminary check and does not validate against the payload
var eventRegex *regexp.Regexp

// If param fields do not match this regex, their syntax is malformed or
// they do not refer to param within the TriggerTemplate
var paramsRegex *regexp.Regexp

func init() {
	eventRegex = regexp.MustCompile(createInterpolationRegex(EventInterpolationBase, true, true))
	paramsRegex = regexp.MustCompile(createInterpolationRegex(ParamsInterpolationBase, false, false))
}

// ValidEventBinding validates the interpolation used by TriggerBindings
func ValidEventBinding(path string) bool {
	return eventRegex.MatchString(path)
}

// ValidTemplateBinding validates the interpolation used by TriggerTemplates
func ValidTemplateBinding(path string) bool {
	return paramsRegex.MatchString(path)
}

// createInterpolationRegex creates an interpolation regex using the specified parameters
// If baseAccess and recursiveAccess are both false, the desired regex will match 1 field deep
func createInterpolationRegex(interpolationBase string, baseAccess bool, recursiveAccess bool) string {
	var fieldAccessRegex string
	if baseAccess && recursiveAccess {
		fieldAccessRegex = fmt.Sprintf("(\\.%s)*", interpolationFieldRegexString)
	} else if recursiveAccess {
		fieldAccessRegex = fmt.Sprintf("(\\.%s)+", interpolationFieldRegexString)
	} else if !baseAccess && !recursiveAccess {
		fieldAccessRegex = fmt.Sprintf("(\\.%s){1}", interpolationFieldRegexString)
	}
	baseAndFieldRegex := interpolationBase + fieldAccessRegex
	interpolatedRegex := strings.Replace(interpolationRegex, InterpolationBaseMock, baseAndFieldRegex, 1)
	return interpolatedRegex
}

// Interpolate interpolates the path without validation
func Interpolate(path string) string {
	return strings.Replace(InterpolationSyntax, InterpolationBaseMock, path, 1)
}
