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

package server

import (
	"context"
	"log"

	"github.com/golang/protobuf/ptypes/empty"
	"gorm.io/gorm"

	"github.com/google/cel-go/cel"
	celenv "github.com/tektoncd/results/pkg/api/server/cel"
	"github.com/tektoncd/results/pkg/api/server/db"
	"github.com/tektoncd/results/pkg/api/server/db/errors"
	"github.com/tektoncd/results/pkg/api/server/db/pagination"
	"github.com/tektoncd/results/pkg/api/server/v1alpha2/auth"
	"github.com/tektoncd/results/pkg/api/server/v1alpha2/result"
	"github.com/tektoncd/results/pkg/internal/protoutil"
	pb "github.com/tektoncd/results/proto/v1alpha2/results_go_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateResult creates a new result in the database.
func (s *Server) CreateResult(ctx context.Context, req *pb.CreateResultRequest) (*pb.Result, error) {
	r := req.GetResult()

	// Validate the incoming request
	parent, _, err := result.ParseName(r.GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if req.GetParent() != parent {
		return nil, status.Error(codes.InvalidArgument, "requested parent does not match resource name")
	}
	if err := s.auth.Check(ctx, parent, auth.ResourceResults, auth.PermissionCreate); err != nil {
		return nil, err
	}

	// Populate Result with server provided fields.
	protoutil.ClearOutputOnly(r)
	r.Id = uid()
	r.CreatedTime = timestamppb.New(clock.Now())
	r.UpdatedTime = timestamppb.New(clock.Now())

	store, err := result.ToStorage(r)
	if err != nil {
		return nil, err
	}

	if err := result.UpdateEtag(store); err != nil {
		return nil, err
	}

	if err := errors.Wrap(s.db.WithContext(ctx).Create(store).Error); err != nil {
		return nil, err
	}
	return result.ToAPI(store), nil
}

// GetResult returns a single Result.
func (s *Server) GetResult(ctx context.Context, req *pb.GetResultRequest) (*pb.Result, error) {
	parent, name, err := result.ParseName(req.GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.auth.Check(ctx, parent, auth.ResourceResults, auth.PermissionGet); err != nil {
		return nil, err
	}
	store, err := getResultByParentName(s.db, parent, name)
	if err != nil {
		return nil, err
	}
	return result.ToAPI(store), nil
}

// UpdateResult updates a Result in the database.
func (s *Server) UpdateResult(ctx context.Context, req *pb.UpdateResultRequest) (*pb.Result, error) {
	// Retrieve result from database by name
	parent, name, err := result.ParseName(req.GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.auth.Check(ctx, parent, auth.ResourceResults, auth.PermissionUpdate); err != nil {
		return nil, err
	}

	var out *pb.Result
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		prev, err := getResultByParentName(tx, parent, name)
		if err != nil {
			return status.Errorf(codes.NotFound, "failed to find a result: %v", err)
		}

		// If the user provided the Etag field, then make sure the value of this field matches what saved in the database.
		// See https://google.aip.dev/154 for more information.
		if req.GetEtag() != "" && req.GetEtag() != prev.Etag {
			return status.Error(codes.FailedPrecondition, "the etag mismatches")
		}

		prev.Annotations = req.GetResult().GetAnnotations()
		prev.UpdatedTime = clock.Now()

		if err := result.UpdateEtag(prev); err != nil {
			return err
		}

		// Write result back to database.
		if err = errors.Wrap(tx.Save(&prev).Error); err != nil {
			log.Printf("failed to save result into database: %v", err)
			return err
		}

		out = result.ToAPI(prev)
		return nil
	})
	return out, err
}

// DeleteResult deletes a given result.
func (s *Server) DeleteResult(ctx context.Context, req *pb.DeleteResultRequest) (*empty.Empty, error) {
	parent, name, err := result.ParseName(req.GetName())
	if err != nil {
		return nil, err
	}
	if err := s.auth.Check(ctx, parent, auth.ResourceResults, auth.PermissionDelete); err != nil {
		return nil, err
	}

	// First get the current result. This ensures that we return NOT_FOUND if
	// the entry is already deleted.
	// This does not need to be done in the same transaction as the delete,
	// since the identifiers are immutable.
	r := &db.Result{}
	get := s.db.WithContext(ctx).
		Where(&db.Result{Parent: parent, Name: name}).
		First(r)
	if err := errors.Wrap(get.Error); err != nil {
		return &empty.Empty{}, err
	}

	// Delete the result.
	delete := s.db.WithContext(ctx).Delete(&db.Result{}, r)
	return &empty.Empty{}, errors.Wrap(delete.Error)
}

func (s *Server) ListResults(ctx context.Context, req *pb.ListResultsRequest) (*pb.ListResultsResponse, error) {
	if req.GetParent() == "" {
		return nil, status.Error(codes.InvalidArgument, "parent missing")
	}
	if err := s.auth.Check(ctx, req.GetParent(), auth.ResourceResults, auth.PermissionList); err != nil {
		return nil, err
	}

	userPageSize, err := pageSize(int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}

	start, err := pageStart(req.GetPageToken(), req.GetFilter())
	if err != nil {
		return nil, err
	}

	prg, err := celenv.ParseFilter(s.env, req.GetFilter())
	if err != nil {
		return nil, err
	}
	// Fetch n+1 items to get the next token.
	out, err := s.getFilteredPaginatedResults(ctx, req.GetParent(), start, userPageSize+1, prg)
	if err != nil {
		return nil, err
	}

	// If we returned the full n+1 items, use the last element as the next page
	// token.
	var nextToken string
	if len(out) > userPageSize {
		next := out[len(out)-1]
		var err error
		nextToken, err = pagination.EncodeToken(next.GetId(), req.GetFilter())
		if err != nil {
			return nil, err
		}
		out = out[:len(out)-1]
	}

	return &pb.ListResultsResponse{
		Results:       out,
		NextPageToken: nextToken,
	}, nil
}

// getFilteredPaginatedResults returns the specified number of results that
// match the given CEL program.
func (s *Server) getFilteredPaginatedResults(ctx context.Context, parent string, start string, pageSize int, prg cel.Program) ([]*pb.Result, error) {
	out := make([]*pb.Result, 0, pageSize)
	batcher := pagination.NewBatcher(pageSize, minPageSize, maxPageSize)
	for len(out) < pageSize {
		batchSize := batcher.Next()
		dbresults := make([]*db.Result, 0, batchSize)
		q := s.db.WithContext(ctx).
			Where("parent = ? AND id > ?", parent, start).
			Limit(batchSize).
			Find(&dbresults)
		if err := errors.Wrap(q.Error); err != nil {
			return nil, err
		}

		// Only return results that match the filter.
		for _, r := range dbresults {
			api := result.ToAPI(r)
			ok, err := result.Match(api, prg)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}

			out = append(out, api)
			if len(out) >= pageSize {
				return out, nil
			}
		}

		// We fetched less results than requested - this means we've exhausted
		// all items.
		if len(dbresults) < batchSize {
			break
		}

		// Set params for next batch.
		start = dbresults[len(dbresults)-1].ID
		batcher.Update(len(dbresults), batchSize)

	}
	return out, nil
}

func getResultByParentName(gdb *gorm.DB, parent, name string) (*db.Result, error) {
	r := &db.Result{}
	if err := errors.Wrap(gdb.Where(&db.Result{Parent: parent, Name: name}).First(r).Error); err != nil {
		return nil, status.Errorf(status.Code(err), "failed to query on database: %v", err)
	}
	return r, nil
}
