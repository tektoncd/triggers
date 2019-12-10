package template

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"

	"k8s.io/client-go/third_party/forked/golang/template"
	"k8s.io/client-go/util/jsonpath"
)

var (
	// tektonVar captures strings that are enclosed in $()
	tektonVar = regexp.MustCompile(`\$\(?([^\)]+)\)`)

	// jsonRegexp is a regular expression for JSONPath expressions
	// with or without the enclosing {} and the leading . inside the curly
	// braces e.g.  'a.b' or '.a.b' or '{a.b}' or '{.a.b}'
	jsonRegexp = regexp.MustCompile(`^\{\.?([^{}]+)\}$|^\.?([^{}]+)$`)
)

// ParseJSONPath extracts a subset of the given JSON input
// using the provided JSONPath expression.
func ParseJSONPath(input interface{}, expr string) (string, error) {
	j := jsonpath.New("").AllowMissingKeys(false)
	buf := new(bytes.Buffer)

	//First turn the expression into fully valid JSONPath
	expr, err := TektonJSONPathExpression(expr)
	if err != nil {
		return "", err
	}

	if err := j.Parse(expr); err != nil {
		return "", err
	}

	fullResults, err := j.FindResults(input)
	if err != nil {
		return "", err
	}

	for _, r := range fullResults {
		if err := printResults(buf, r); err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}

// PrintResults writes the results into writer
// This is a slightly modified copy of the original
// j.PrintResults from k8s.io/client-go/util/jsonpath/jsonpath.go
// in that it uses calls `textValue()` for instead of `evalToText`
// This is a workaround for kubernetes/kubernetes#16707
func printResults(wr io.Writer, results []reflect.Value) error {
	for i, r := range results {
		text, err := textValue(r)
		if err != nil {
			return err
		}
		if i != len(results)-1 {
			text = append(text, ' ')
		}
		if _, err := wr.Write(text); err != nil {
			return err
		}
	}
	return nil
}

// textValue translates reflect value to corresponding text
// If the value if an array or map, it returns a JSON representation
// of the value (as opposed to the internal go representation of the value)
// Otherwise, the text value is from the `evalToText` function, originally from
// k8s.io/client-go/util/jsonpath/jsonpath.go
func textValue(v reflect.Value) ([]byte, error) {
	t := reflect.TypeOf(v.Interface())
	// special case for null values in JSON; evalToText() returns <nil> here
	if t == nil {
		return []byte("null"), nil
	}

	switch t.Kind() {
	// evalToText() returns <map> ....; return JSON string instead.
	case reflect.Map, reflect.Slice:
		return json.Marshal(v.Interface())
	default:
		return evalToText(v)
	}
}

// evalToText translates reflect value to corresponding text
// This is a unmodified copy of j.evalToText from k8s.io/client-go/util/jsonpath/jsonpath.go
func evalToText(v reflect.Value) ([]byte, error) {
	iface, ok := template.PrintableValue(v)
	if !ok {
		// only happens if v is a Chan or a Func
		return nil, fmt.Errorf("can't print type %s", v.Type())
	}
	var buffer bytes.Buffer
	fmt.Fprint(&buffer, iface)
	return buffer.Bytes(), nil
}

// TektonJSONPathExpression returns a valid JSONPath expression. It accepts
// a "RelaxedJSONPath" expression that is wrapped in the Tekton variable
// interpolation syntax i.e. $(). RelaxedJSONPath expressions can optionally
// omit the leading curly braces '{}' and '.'
func TektonJSONPathExpression(expr string) (string, error) {
	if !isTektonExpr(expr) {
		return "", errors.New("expression not wrapped in $()")
	}
	unwrapped := strings.TrimSuffix(strings.TrimPrefix(expr, "$("), ")")
	return relaxedJSONPathExpression(unwrapped)
}

// RelaxedJSONPathExpression attempts to be flexible with JSONPath expressions, it accepts:
//   * metadata.name (no leading '.' or curly braces '{...}'
//   * {metadata.name} (no leading '.')
//   * .metadata.name (no curly braces '{...}')
//   * {.metadata.name} (complete expression)
// And transforms them all into a valid jsonpath expression:
//   {.metadata.name}
// This function has been copied as-is from
// https://github.com/kubernetes/kubectl/blob/c273777957bd657233cf867892fb061a6498dab8/pkg/cmd/get/customcolumn.go#L47
func relaxedJSONPathExpression(pathExpression string) (string, error) {
	if len(pathExpression) == 0 {
		return pathExpression, nil
	}
	submatches := jsonRegexp.FindStringSubmatch(pathExpression)
	if submatches == nil {
		return "", fmt.Errorf("unexpected path string, expected a 'name1.name2' or '.name1.name2' or '{name1.name2}' or '{.name1.name2}'")
	}
	if len(submatches) != 3 {
		return "", fmt.Errorf("unexpected submatch list: %v", submatches)
	}
	var fieldSpec string
	if len(submatches[1]) != 0 {
		fieldSpec = submatches[1]
	} else {
		fieldSpec = submatches[2]
	}
	return fmt.Sprintf("{.%s}", fieldSpec), nil
}

// IsTektonExpr returns true if the expr is wrapped in $()
func isTektonExpr(expr string) bool {
	return tektonVar.MatchString(expr)
}
