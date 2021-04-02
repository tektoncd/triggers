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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/cel-go/cel"
	celenv "github.com/tektoncd/results/pkg/api/server/cel"
	"github.com/tektoncd/results/pkg/api/server/db"
	"github.com/tektoncd/results/pkg/api/server/db/errors"
	"github.com/tektoncd/results/pkg/api/server/db/pagination"
	"github.com/tektoncd/results/pkg/api/server/v1alpha2/auth"
	"github.com/tektoncd/results/pkg/api/server/v1alpha2/record"
	"github.com/tektoncd/results/pkg/api/server/v1alpha2/result"
	"github.com/tektoncd/results/pkg/internal/protoutil"
	pb "github.com/tektoncd/results/proto/v1alpha2/results_go_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func (s *Server) CreateRecord(ctx context.Context, req *pb.CreateRecordRequest) (*pb.Record, error) {
	r := req.GetRecord()

	// Validate the incoming request
	parent, resultName, name, err := record.ParseName(r.GetName())
	if err != nil {
		return nil, err
	}
	if req.GetParent() != result.FormatName(parent, resultName) {
		return nil, status.Error(codes.InvalidArgument, "requested parent does not match resource name")
	}
	if err := s.auth.Check(ctx, parent, auth.ResourceRecords, auth.PermissionCreate); err != nil {
		return nil, err
	}

	// Look up the result ID from the name. This does not have to happen
	// transactionally with the insert since name<->ID mappings are immutable,
	// and if the the parent result is deleted mid-request, the insert should
	// fail due to foreign key constraints.
	resultID, err := s.getResultID(ctx, parent, resultName)
	if err != nil {
		return nil, err
	}

	// Populate Result with server provided fields.
	protoutil.ClearOutputOnly(r)
	r.Id = uid()
	r.CreatedTime = timestamppb.New(clock.Now())
	r.UpdatedTime = timestamppb.New(clock.Now())

	store, err := record.ToStorage(parent, resultName, resultID, name, req.GetRecord())
	if err != nil {
		return nil, err
	}
	if err := record.UpdateEtag(store); err != nil {
		return nil, err
	}
	q := s.db.WithContext(ctx).
		Model(store).
		Create(store).Error
	if err := errors.Wrap(q); err != nil {
		return nil, err
	}

	return record.ToAPI(store)
}

// resultID is a utility struct to extract partial Result data representing
// Result name <-> ID mappings.
type resultID struct {
	Name string
	ID   string
}

func (s *Server) getResultIDImpl(ctx context.Context, parent, result string) (string, error) {
	id := new(resultID)
	q := s.db.WithContext(ctx).
		Model(&db.Result{}).
		Where(&db.Result{Parent: parent, Name: result}).
		First(id)
	if err := errors.Wrap(q.Error); err != nil {
		return "", err
	}
	return id.ID, nil
}

// GetRecord returns a single Record.
func (s *Server) GetRecord(ctx context.Context, req *pb.GetRecordRequest) (*pb.Record, error) {
	parent, result, name, err := record.ParseName(req.GetName())
	if err != nil {
		return nil, err
	}
	if err := s.auth.Check(ctx, parent, auth.ResourceRecords, auth.PermissionGet); err != nil {
		return nil, err
	}

	r, err := getRecord(s.db.WithContext(ctx), parent, result, name)
	if err != nil {
		return nil, err
	}
	return record.ToAPI(r)
}

func getRecord(txn *gorm.DB, parent, result, name string) (*db.Record, error) {
	store := &db.Record{}
	q := txn.
		Where(&db.Record{Result: db.Result{Parent: parent, Name: result}, Name: name}).
		First(store)
	if err := errors.Wrap(q.Error); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Server) ListRecords(ctx context.Context, req *pb.ListRecordsRequest) (*pb.ListRecordsResponse, error) {
	if req.GetParent() == "" {
		return nil, status.Error(codes.InvalidArgument, "parent missing")
	}
	if err := s.auth.Check(ctx, req.GetParent(), auth.ResourceRecords, auth.PermissionList); err != nil {
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
	out, err := s.getFilteredPaginatedRecords(ctx, req.GetParent(), start, userPageSize+1, prg)
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

	return &pb.ListRecordsResponse{
		Records:       out,
		NextPageToken: nextToken,
	}, nil
}

// getFilteredPaginatedRecords returns the specified number of results that
// match the given CEL program.
func (s *Server) getFilteredPaginatedRecords(ctx context.Context, parent, start string, pageSize int, prg cel.Program) ([]*pb.Record, error) {
	parent, result, err := result.ParseName(parent)
	if err != nil {
		return nil, err
	}

	out := make([]*pb.Record, 0, pageSize)
	batcher := pagination.NewBatcher(pageSize, minPageSize, maxPageSize)
	for len(out) < pageSize {
		batchSize := batcher.Next()
		dbrecords := make([]*db.Record, 0, batchSize)
		q := s.db.WithContext(ctx).Where("parent = ? AND id > ?", parent, start)
		// Specifying `-` allows users to read Records across Results.
		// See https://google.aip.dev/159 for more details.
		if result != "-" {
			q = q.Where("result_name = ?", result)
		}
		q = q.Limit(batchSize).Find(&dbrecords)
		if err := errors.Wrap(q.Error); err != nil {
			return nil, err
		}

		// Only return results that match the filter.
		for _, r := range dbrecords {
			api, err := record.ToAPI(r)
			if err != nil {
				return nil, err
			}
			ok, err := record.Match(api, prg)
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
		if len(dbrecords) < batchSize {
			break
		}

		// Set params for next batch.
		start = dbrecords[len(dbrecords)-1].ID
		batcher.Update(len(dbrecords), batchSize)
	}
	return out, nil
}

// UpdateRecord updates a record in the database.
func (s *Server) UpdateRecord(ctx context.Context, req *pb.UpdateRecordRequest) (*pb.Record, error) {
	in := req.GetRecord()

	parent, result, name, err := record.ParseName(in.GetName())
	if err != nil {
		return nil, err
	}
	if err := s.auth.Check(ctx, parent, auth.ResourceRecords, auth.PermissionUpdate); err != nil {
		return nil, err
	}

	protoutil.ClearOutputOnly(in)

	var out *pb.Record
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		r, err := getRecord(tx, parent, result, name)
		if err != nil {
			return err
		}

		// If the user provided the Etag field, then make sure the value of this field matches what saved in the database.
		// See https://google.aip.dev/154 for more information.
		if req.GetEtag() != "" && req.GetEtag() != r.Etag {
			return status.Error(codes.FailedPrecondition, "the etag mismatches")
		}

		// Merge existing data with user request.
		pb, err := record.ToAPI(r)
		if err != nil {
			return err
		}
		// TODO: field mask support.
		proto.Merge(pb, in)

		pb.UpdatedTime = timestamppb.New(clock.Now())

		// Convert back to storage and store.
		s, err := record.ToStorage(r.Parent, r.ResultName, r.ResultID, r.Name, pb)
		if err != nil {
			return err
		}
		if err := record.UpdateEtag(s); err != nil {
			return err
		}
		if err := errors.Wrap(tx.Save(s).Error); err != nil {
			return err
		}

		pb.Etag = s.Etag
		out = pb
		return nil
	})
	return out, err
}

// DeleteRecord deletes a given record.
func (s *Server) DeleteRecord(ctx context.Context, req *pb.DeleteRecordRequest) (*empty.Empty, error) {
	parent, result, name, err := record.ParseName(req.GetName())
	if err != nil {
		return nil, err
	}
	if err := s.auth.Check(ctx, parent, auth.ResourceRecords, auth.PermissionDelete); err != nil {
		return &empty.Empty{}, err
	}

	// First get the current record. This ensures that we return NOT_FOUND if
	// the entry is already deleted.
	// This does not need to be done in the same transaction as the delete,
	// since the identifiers are immutable.
	r, err := getRecord(s.db, parent, result, name)
	if err != nil {
		return &empty.Empty{}, err
	}
	return &empty.Empty{}, errors.Wrap(s.db.WithContext(ctx).Delete(&db.Record{}, r).Error)
}
