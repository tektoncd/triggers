package json_test

import (
	"bytes"
	gojson "encoding/json"
	"io/ioutil"
	"fmt"
	"github.com/tektoncd/triggers/pkg/parse"
	. "github.com/tektoncd/triggers/pkg/parse/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFieldsFromByteMessage(t *testing.T) {
	var testNames []string
	var payloads [][]byte
	// Populates payloads using examples
	err := filepath.Walk("./examples", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.Errorf("Failure accessing path %q: %v\n", path, err)
			return err
		}
		if !info.IsDir() {
			t.Logf("Reading %s",path)
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			payloads = append(payloads,b)
			fileNameTrimmed := strings.TrimSuffix(path,".json")
			testNames = append(testNames, strings.Replace(strings.Title(fileNameTrimmed),"/","",-1))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Unable to load example payloads: %v",err)
	}

	// Validate all fields can be pulled for each respective payload
	for i := 0; i < len(payloads); i++ {
		iCopy := i
		t.Run(testNames[i],func(t *testing.T) {
			// Grab bindings and expected values
			eventBindings, expectedValues, err := eventBindingDigger(t, payloads[iCopy], parse.EventInterpolationBase)
			if err != nil {
				t.Errorf("Failed to generate event bindings for %s payload",testNames[iCopy])
				return
			}
			// Grab actual values
			actualValues := FieldsFromByteMessage(payloads[iCopy], eventBindings)
			for j := range actualValues {
				validationError := fmt.Sprintf("Failed to validate %s binding:",eventBindings[j])
				// Remove spacing to normalize both values
				actualBuffer := new(bytes.Buffer)
				expectedBuffer := new(bytes.Buffer)
				if err := gojson.Compact(actualBuffer, []byte(actualValues[j])); err != nil {
					t.Errorf("%s Unable to compact actual bytes",validationError)
					continue
				}
				if err := gojson.Compact(expectedBuffer, []byte(expectedValues[j])); err != nil {
					t.Errorf("%s Unable to compact expected bytes",validationError)
					continue
				}
				// Grab token to determine value type
				expectedBufferCopy := bytes.NewBuffer(expectedBuffer.Bytes())
				decoder := gojson.NewDecoder(expectedBufferCopy)
				token, err := decoder.Token()
				if err != nil {
					t.Errorf("%s Error getting token from expected bytes",validationError)
					continue
				}
				// Validate the binding maps to the value as intended
				_, ok := token.(gojson.Delim) // Detect whether the binding is a json structure or primitive
				if ok { // Compare json structures
					var expected, actual interface{}
					err := gojson.Unmarshal(expectedBuffer.Bytes(), &expected)
					if err != nil {
						t.Errorf("%s Failed to unmarshal expected value for comparison\nError: %v", validationError, err)
					}
					err = gojson.Unmarshal(actualBuffer.Bytes(), &actual)
					if err != nil {
						t.Fatalf("%s Failed to unmarshal actual value for comparison\nError: %v", validationError, err)
					}
					if !reflect.DeepEqual(expected, actual) {
						t.Errorf("%s \nExpected: %v\nActual: %v", validationError, expected, actual)
					}
				} else { // Compare json key/primitive
					expected, actual := expectedBuffer.String(), actualBuffer.String()
					if expected != actual {
						t.Errorf("%s \nExpected: %v\nActual: %v", validationError, expected, actual)
					}
				}
			}
		})
	}
}

// TestEventBindingDigger tests eventBindingDigger to ensure all possible bindings are returned using the simple digger example payload.
// The simpel digger example payload is space compacted
func TestEventBindingDigger(t *testing.T) {
	// Small example file/payload used to validate the functionality of eventBindingDigger
	filePath := "examples/digger/digger.json"
	t.Logf("Reading %s",filePath)
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Error(err)
	}
	// Create assertion map
	// The example digger payload fields have been predefined as space compacted
	bindingValueMap := map[string]string {
		parse.Interpolate("event"): string(b),
		parse.Interpolate("event.a"): `"a"`,
		parse.Interpolate("event.b"): `true`,
		parse.Interpolate("event.c"): `30`,
		parse.Interpolate("event.d"): `[]`,
		parse.Interpolate("event.e"): `["a"]`,
		parse.Interpolate("event.f"): `[false]`,
		parse.Interpolate("event.g"): `[3]`,
		parse.Interpolate("event.h"): `[{"a":"a"}]`,
		parse.Interpolate("event.i"): `{"a":"a"}`,
		parse.Interpolate("event.i.a"): `"a"`,
	}
	// Grab values
	eventBindings, actualValues, err := eventBindingDigger(t,b,parse.EventInterpolationBase)
	if err != nil {
		t.Fatalf("Failed to generate event bindings for %s payload",filePath)
	}
	
	// Ensure sizing is the same 
	if len(eventBindings) != len(bindingValueMap) {
		t.Fatalf("Length of eventBindings[%d] did not match expected[%d]",len(eventBindings),len(bindingValueMap))
	}
	if len(actualValues) != len(bindingValueMap) {
		t.Fatalf("Length of actualValues[%d] did not match expected[%d]",len(actualValues),len(bindingValueMap))
	}
	// Validate against assertion map
	for i := 0; i< len(eventBindings);i++ {
		expectedValue := bindingValueMap[eventBindings[i]]
		actualValue := actualValues[i]
		if actualValue != expectedValue {
			t.Fatalf("Actual value %s did not match expected %s",actualValue,expectedValue)
		}
	}
}

// eventBindingDigger returns all possible event bindings and corresponding expected values from the payload
// Digs into map recursively whenever the value is a json object
// Inherent to the marshalling of json, expectedValues cannot by guaranteed in the same order as payload
func eventBindingDigger(t *testing.T, payload []byte, location string) (eventBindings []string, expectedValues []string, err error) {
	// Add the entire event payload/base
	eventBindings = append(eventBindings, parse.Interpolate(location))
	expectedValues = append(expectedValues, string(payload))

	// Store event as map to make it iterable
	m := make(map[string]interface{})
	err = gojson.Unmarshal(payload, &m)
	if err != nil {
		return nil, nil, err
	}
	// Iterate over fields (potentially recursively) to capture all event bindings and expected values
	for field, value := range m {
		currentLocation := fmt.Sprintf("%s.%s", location, field)
		b, err := gojson.Marshal(value)
		if err != nil {
			t.Fatalf("Failed to marshal value %v", value)
		}
		nestedMap, ok := value.(map[string]interface{})
		if ok {
			nestedBytes, _ := gojson.Marshal(&nestedMap) 
			nestedBindings, nestedValues, _ := eventBindingDigger(t, nestedBytes, currentLocation)
			eventBindings, expectedValues = append(eventBindings, nestedBindings...), append(expectedValues, nestedValues...)
		} else {
			eventBindings = append(eventBindings, parse.Interpolate(currentLocation))
			expectedValues = append(expectedValues, string(b))
		}
	}
	return eventBindings, expectedValues, nil
}