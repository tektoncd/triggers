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

// Package record provides utilities for manipulating and validating Records.
package record

import (
	"fmt"
	"regexp"

	"github.com/google/cel-go/cel"
	resultscel "github.com/tektoncd/results/pkg/api/server/cel"
	"github.com/tektoncd/results/pkg/api/server/db"
	pb "github.com/tektoncd/results/proto/v1alpha2/results_go_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	// NameRegex matches valid name specs for a Result.
	NameRegex = regexp.MustCompile("(^[a-z0-9_-]{1,63})/results/([a-z0-9_-]{1,63})/records/([a-z0-9_-]{1,63}$)")
)

// ParseName splits a full Result name into its individual (parent, result, name)
// components.
func ParseName(raw string) (parent, result, name string, err error) {
	s := NameRegex.FindStringSubmatch(raw)
	if len(s) != 4 {
		return "", "", "", status.Errorf(codes.InvalidArgument, "name must match %s", NameRegex.String())
	}
	return s[1], s[2], s[3], nil
}

// FormatName takes in a parent ("a/results/b") and record name ("c") and
// returns the full resource name ("a/results/b/records/c").
func FormatName(parent, name string) string {
	return fmt.Sprintf("%s/records/%s", parent, name)
}

// ToStorage converts an API Record into its corresponding database storage
// equivalent.
// parent,result,name should be the name parts (e.g. not containing "/results/" or "/records/").
func ToStorage(parent, resultName, resultID, name string, r *pb.Record) (*db.Record, error) {
	data, err := proto.Marshal(r.Data)
	if err != nil {
		return nil, err
	}
	dbr := &db.Record{
		Parent:     parent,
		ResultName: resultName,
		ResultID:   resultID,

		ID:   r.GetId(),
		Name: name,

		Data: data,
		Etag: r.Etag,
	}
	if r.CreatedTime.IsValid() {
		dbr.CreatedTime = r.CreatedTime.AsTime()
	}
	if r.UpdatedTime.IsValid() {
		dbr.UpdatedTime = r.UpdatedTime.AsTime()
	}
	return dbr, nil
}

// ToAPI converts a database storage Record into its corresponding API
// equivalent.
func ToAPI(r *db.Record) (*pb.Record, error) {
	out := &pb.Record{
		Name: fmt.Sprintf("%s/results/%s/records/%s", r.Parent, r.ResultName, r.Name),
		Id:   r.ID,
		Etag: r.Etag,
	}

	if !r.CreatedTime.IsZero() {
		out.CreatedTime = timestamppb.New(r.CreatedTime)
	}
	if !r.UpdatedTime.IsZero() {
		out.UpdatedTime = timestamppb.New(r.UpdatedTime)
	}

	// Check if data was stored before unmarshalling, to avoid returning `{}`.
	if r.Data != nil {
		any := new(anypb.Any)
		if err := proto.Unmarshal(r.Data, any); err != nil {
			return nil, err
		}
		out.Data = any
	}

	return out, nil
}

// Match determines whether the given CEL filter matches the result.
func Match(r *pb.Record, prg cel.Program) (bool, error) {
	if r == nil {
		return false, nil
	}
	return resultscel.Match(prg, "record", r)
}

// UpdateEtag updates the etag field of a record according to its content.
// The record should at least have its `Id` and `UpdatedTime` fields set.
func UpdateEtag(r *db.Record) error {
	if r.ID == "" {
		return fmt.Errorf("the ID field must be set")
	}
	if r.UpdatedTime.IsZero() {
		return status.Error(codes.Internal, "the UpdatedTime field must be set")
	}
	r.Etag = fmt.Sprintf("%s-%v", r.ID, r.UpdatedTime.UnixNano())
	return nil
}
