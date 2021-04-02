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

// Package sqlite provides sqlite-specific error checking. This is
// purposefully broken out from the rest of the errors package so that we can
// isolate go-sqlite3's cgo dependency away from the main MySQL based library
// to simplify our testing + deployment.
package sqlite

import (
	"github.com/mattn/go-sqlite3"
	"github.com/tektoncd/results/pkg/api/server/db/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// sqlite converts sqlite3 error codes to gRPC status codes. This is not an
// exhaustive list.
// See https://pkg.go.dev/github.com/mattn/go-sqlite3#pkg-variables for list of
// error codes.
func sqlite(err error) codes.Code {
	serr, ok := err.(sqlite3.Error)
	if !ok {
		return status.Code(err)
	}

	switch serr.Code {
	case sqlite3.ErrConstraint:
		switch serr.ExtendedCode {
		case sqlite3.ErrConstraintUnique:
			return codes.AlreadyExists
		case sqlite3.ErrConstraintForeignKey:
			return codes.FailedPrecondition
		}
		return codes.InvalidArgument
	case sqlite3.ErrNotFound:
		return codes.NotFound
	}
	return status.Code(err)
}

func init() {
	errors.RegisterErrorSpace(sqlite)
}
