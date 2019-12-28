package template

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var objects = `{"a":"v\r\n烈","c":{"d":"e"},"empty": "","null": null, "number": 42}`
var arrays = `[{"a": "b"}, {"c": "d"}, {"e": "f"}]`

// Checks that we print JSON strings when the JSONPath selects
// an array or map value and regular values otherwise
func TestParseJSONPath(t *testing.T) {
	var objectBody = fmt.Sprintf(`{"body":%s}`, objects)
	tests := []struct {
		name string
		expr string
		in   string
		want string
	}{{
		name: "objects",
		in:   objectBody,
		expr: "$(body)",
		// TODO: Do we need to escape backslashes for backwards compat?
		want: objects,
	}, {
		name: "array of objects",
		in:   fmt.Sprintf(`{"body":%s}`, arrays),
		expr: "$(body)",
		want: arrays,
	}, {
		name: "array of values",
		in:   `{"body": ["a", "b", "c"]}`,
		expr: "$(body)",
		want: `["a", "b", "c"]`,
	}, {
		name: "string values",
		in:   objectBody,
		expr: "$(body.a)",
		want: "v\\r\\n烈",
	}, {
		name: "empty string",
		in:   objectBody,
		expr: "$(body.empty)",
		want: "",
	}, {
		name: "numbers",
		in:   objectBody,
		expr: "$(body.number)",
		want: "42",
	}, {
		name: "booleans",
		in:   `{"body": {"bool": true}}`,
		expr: "$(body.bool)",
		want: "true",
	}, {
		name: "null values",
		in:   objectBody,
		expr: "$(body.null)",
		want: "null",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data interface{}
			err := json.Unmarshal([]byte(tt.in), &data)
			if err != nil {
				t.Fatalf("Could not unmarshall body : %q", err)
			}
			got, err := ParseJSONPath(data, tt.expr)
			if err != nil {
				t.Fatalf("ParseJSONPath() error = %v", err)
			}
			if diff := cmp.Diff(strings.Replace(tt.want, " ", "", -1), got); diff != "" {
				t.Errorf("ParseJSONPath() -want,+got: %s", diff)
			}
		})
	}
}

func TestParseJSONPath_Error(t *testing.T) {
	testJSON := `{"body": {"key": "val"}}`
	invalidExprs := []string{
		"$({.hello)",
		"$(+12.3.0)",
		"$([1)",
		"$(body",
		"body)",
		"body",
		"$(body.missing)",
		"$(body.key[0])",
	}
	var data interface{}
	err := json.Unmarshal([]byte(testJSON), &data)
	if err != nil {
		t.Fatalf("Could not unmarshall body : %q", err)
	}

	for _, expr := range invalidExprs {
		t.Run(expr, func(t *testing.T) {
			got, err := ParseJSONPath(data, expr)
			if err == nil {
				t.Errorf("ParseJSONPath() did not return expected error; got = %v", got)
			}
		})
	}
}

func TestTektonJSONPathExpression(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{"$(metadata.name)", "{.metadata.name}"},
		{"$(.metadata.name)", "{.metadata.name}"},
		{"$({.metadata.name})", "{.metadata.name}"},
		{"$()", ""},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := TektonJSONPathExpression(tt.expr)
			if err != nil {
				t.Errorf("TektonJSONPathExpression() unexpected error = %v,  got = %v", err, got)
			}
			if got != tt.want {
				t.Errorf("TektonJSONPathExpression() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTektonJSONPathExpression_Error(t *testing.T) {
	tests := []string{
		"{.metadata.name}", // not wrapped in $()
		"",
		"$({asd)",
		"$({)",
		"$({foo.bar)",
		"$(foo.bar})",
		"$({foo.bar}})",
		"$({{foo.bar)",
	}
	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			_, err := TektonJSONPathExpression(expr)
			if err == nil {
				t.Errorf("TektonJSONPathExpression() did not get expected error for expression = %s", expr)
			}
		})
	}
}

func TestRelaxedJSONPathExpression(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{"metadata.name", "{.metadata.name}"},
		{".metadata.name", "{.metadata.name}"},
		{"{.metadata.name}", "{.metadata.name}"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := relaxedJSONPathExpression(tt.expr)
			if err != nil {
				t.Errorf("TektonJSONPathExpression() unexpected error = %v,  got = %v", err, got)
			}
			if got != tt.want {
				t.Errorf("TektonJSONPathExpression() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRelaxedJSONPathExpression_Error(t *testing.T) {
	tests := []string{
		"{foo.bar",
		"foo.bar}",
		"{foo.bar}}",
		"{{foo.bar}",
	}
	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			got, err := relaxedJSONPathExpression(expr)
			if err == nil {
				t.Errorf("TektonJSONPathExpression() did not get expected error = %v,  got = %v", err, got)
			}
		})
	}
}
