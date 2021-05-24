/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package template

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"reflect"
	"regexp"
	"strings"

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

// parseJSONPath extracts a subset of the given JSON input
// using the provided JSONPath expression.
func parseJSONPath(input interface{}, expr string) (string, error) {
	j := jsonpath.New("").AllowMissingKeys(false)
	buf := new(bytes.Buffer)

	// First turn the expression into fully valid JSONPath
	expr, err := tektonJSONPathExpression(expr)
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
func printResults(wr io.Writer, values []reflect.Value) error {
	results, err := getResults(values)
	if err != nil {
		return fmt.Errorf("error getting values for jsonpath results: %w", err)
	}

	if _, err := wr.Write(results); err != nil {
		return err
	}
	return nil
}

func getResults(values []reflect.Value) ([]byte, error) {
	if len(values) == 1 {
		v := values[0]
		t := reflect.TypeOf(v.Interface())
		switch {
		case t == nil:
			return []byte("null"), nil
		case t.Kind() == reflect.String:
			b, err := json.Marshal(v.Interface())
			if err != nil {
				return nil, fmt.Errorf("unable to marshal string value %v: %v", v, err)
			}
			// A valid json string is surrounded by quotation marks; we are using this function to
			// create a representation of the json value that can be embedded in a CRD definition and
			// we want to leave it up to the user if they want the surrounding quotation marks or not.
			return b[1 : len(b)-1], nil
		default:
			return json.Marshal(v.Interface())
		}
	}

	// More than one result - we need to return a JSON array response
	results := []interface{}{}
	for _, r := range values {
		t := reflect.TypeOf(r.Interface())
		if t == nil {
			results = append(results, nil)
		} else {
			// No special case for string here unlike above since its going to be part of a JSON array
			results = append(results, r.Interface())
		}
	}
	return json.Marshal(results)
}

// tektonJSONPathExpression returns a valid JSONPath expression. It accepts
// a "RelaxedJSONPath" expression that is wrapped in the Tekton variable
// interpolation syntax i.e. $(). RelaxedJSONPath expressions can optionally
// omit the leading curly braces '{}' and '.'
func tektonJSONPathExpression(expr string) (string, error) {
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

// findTektonExpressions searches for and returns a slice of
// all substrings that are wrapped in $()
// substring with "header." is converted with CanonicalMIMEHeaderKey in the first array
// the second array has the original substrings
func findTektonExpressions(in string) ([]string, []string) {
	results := []string{}
	originals := []string{}

	// No expressions to return
	if !strings.Contains(in, "$(") {
		return results, originals
	}
	// Splits string on $( to find potential Tekton expressions
	maybeExpressions := strings.Split(in, "$(")
	for _, e := range maybeExpressions[1:] { // Split always returns at least one element
		// Iterate until we find the first unbalanced )
		numOpenBrackets := 0
		for i, ch := range e {
			switch ch {
			case '(':
				numOpenBrackets++
			case ')':
				numOpenBrackets--
				if numOpenBrackets < 0 {
					raw := e[:i]
					originals = append(originals, fmt.Sprintf("$(%s)", raw))
					if strings.Index(raw, "header.") == 0 {
						raw = "header." + textproto.CanonicalMIMEHeaderKey(raw[len("header."):])
					}
					results = append(results, fmt.Sprintf("$(%s)", raw))
				}
			default:
				continue
			}
			if numOpenBrackets < 0 {
				break
			}
		}
	}
	return results, originals
}
