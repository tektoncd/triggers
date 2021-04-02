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

// Package result provides utilities for manipulating and validating Results.
package result

import (
	"fmt"
	"regexp"

	"github.com/google/cel-go/cel"
	resultscel "github.com/tektoncd/results/pkg/api/server/cel"
	"github.com/tektoncd/results/pkg/api/server/db"
	pb "github.com/tektoncd/results/proto/v1alpha2/results_go_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	// NameRegex matches valid name specs for a Result.
	NameRegex = regexp.MustCompile("(^[a-z0-9_-]{1,63})/results/([a-z0-9_-]{1,63}$)")
)

// ParseName splits a full Result name into its individual (parent, name)
// components.
func ParseName(raw string) (parent, name string, err error) {
	s := NameRegex.FindStringSubmatch(raw)
	if len(s) != 3 {
		return "", "", status.Errorf(codes.InvalidArgument, "name must match %s", NameRegex.String())
	}
	return s[1], s[2], nil
}

// FormatName takes in a parent ("a") and result name ("b") and
// returns the full resource name ("a/results/b").
func FormatName(parent, name string) string {
	return fmt.Sprintf("%s/results/%s", parent, name)
}

// ToStorage converts an API Result into its corresponding database storage
// equivalent.
// parent,name should be the name parts (e.g. not containing "/results/").
func ToStorage(r *pb.Result) (*db.Result, error) {
	parent, name, err := ParseName(r.GetName())
	if err != nil {
		return nil, err
	}
	result := &db.Result{
		Parent:      parent,
		ID:          r.GetId(),
		Name:        name,
		UpdatedTime: r.UpdatedTime.AsTime(),
		CreatedTime: r.CreatedTime.AsTime(),
		Annotations: r.Annotations,
		Etag:        r.Etag,
	}
	return result, nil
}

// ToAPI converts a database storage Result into its corresponding API
// equivalent.
func ToAPI(r *db.Result) *pb.Result {
	return &pb.Result{
		Name:        FormatName(r.Parent, r.Name),
		Id:          r.ID,
		CreatedTime: timestamppb.New(r.CreatedTime),
		UpdatedTime: timestamppb.New(r.UpdatedTime),
		Annotations: r.Annotations,
		Etag:        r.Etag,
	}
}

// Match determines whether the given CEL filter matches the result.
func Match(r *pb.Result, prg cel.Program) (bool, error) {
	if r == nil {
		return false, nil
	}
	return resultscel.Match(prg, "result", r)
}

// UpdateEtag updates the etag field of a result according to its content.
// The result should at least have its `Id` and `UpdatedTime` fields set.
func UpdateEtag(r *db.Result) error {
	if r.ID == "" {
		return fmt.Errorf("the ID field must be set")
	}
	if r.UpdatedTime.IsZero() {
		return status.Error(codes.Internal, "the UpdatedTime field must be set")
	}
	r.Etag = fmt.Sprintf("%s-%v", r.ID, r.UpdatedTime.UnixNano())
	return nil
}
