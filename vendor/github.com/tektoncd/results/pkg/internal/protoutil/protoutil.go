// Copyright 2020 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package protoutil provides utilities for manipulating protos in tests.
package protoutil

import (
	"testing"

	fbpb "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// Any wraps a proto message in an Any proto, or causes the test to fail.
func Any(t testing.TB, m proto.Message) *anypb.Any {
	t.Helper()
	a, err := anypb.New(m)
	if err != nil {
		t.Fatalf("error wrapping Any proto: %v", err)
	}
	return a
}

// AnyBytes returns the marshalled bytes of an Any proto wrapping the given
// message, or causes the test to fail.
func AnyBytes(t testing.TB, m proto.Message) []byte {
	t.Helper()
	b, err := proto.Marshal(Any(t, m))
	if err != nil {
		t.Fatalf("error marshalling Any proto: %v", err)
	}
	return b
}

// ClearOutputOnly clears any proto fields marked as OUTPUT_ONLY.
func ClearOutputOnly(pb proto.Message) {
	m := pb.ProtoReflect()
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		opts := fd.Options().(*descriptorpb.FieldOptions)
		for _, b := range proto.GetExtension(opts, fbpb.E_FieldBehavior).([]fbpb.FieldBehavior) {
			if b == fbpb.FieldBehavior_OUTPUT_ONLY {
				m.Clear(fd)
			}
		}
		return true
	})
}
