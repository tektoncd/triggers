package json

import (
	"fmt"
	"github.com/tektoncd/triggers/pkg/parse"
	"github.com/tidwall/gjson"
	"strings"
)

// FieldsFromByteMessage returns all fields as specified by the messageBindings from the event byte message.
// The messageBindings are assumed valid
func FieldsFromByteMessage(event []byte, messageBindings []string) []string {
	fieldValues := make([]string, len(messageBindings))
	for i := range messageBindings {
		fieldValues[i] = FieldFromByteMessage(event, messageBindings[i])
	}
	return fieldValues
}

// FieldFromByteMessage extracts the field as specified by the messageBinding from the event byte message.
// The messageBinding is assumed valid
func FieldFromByteMessage(event []byte, messageBinding string) string {
	prefixAndSuffix := strings.Split(parse.InterpolationSyntax, parse.InterpolationBaseMock)

	// Remove interpolation and EventInterpolationBase prefix
	prefixTrimmedPath := strings.TrimPrefix(messageBinding, fmt.Sprintf("%s%s", prefixAndSuffix[0], parse.EventInterpolationBase))
	// Remove leading dot (when messageBinding is anything other than EventInterpolationBase path)
	prefixTrimmedPath = strings.TrimPrefix(prefixTrimmedPath, ".")
	gjsonPath := strings.TrimSuffix(prefixTrimmedPath, prefixAndSuffix[1])
	if gjsonPath == "" {
		return string(event)
	} else {
		return gjson.GetBytes(event, gjsonPath).Raw
	}
}
