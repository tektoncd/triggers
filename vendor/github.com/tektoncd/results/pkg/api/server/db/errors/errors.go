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

package errors

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

var (
	errorSpace ErrorSpace
)

// ErrorSpace allows implementations to inject database specific error checking
// to the application.
type ErrorSpace func(error) codes.Code

// RegisterErrorSpace registers the ErrorSpace - last one wins.
func RegisterErrorSpace(f ErrorSpace) {
	errorSpace = f
}

// Wrap converts database error codes into their corresponding gRPC status
// codes.
func Wrap(err error) error {
	if err == nil {
		return err
	}

	// Check for gorm provided errors first - these are more likely to be
	// supported across drivers.
	if code, ok := gormCode(err); ok {
		return status.Error(code, err.Error())
	}

	// Fallback to implementation specific codes.
	if errorSpace != nil {
		return status.Error(errorSpace(err), err.Error())
	}

	return err
}

// gormCode returns gRPC status codes corresponding to gorm errors. This is not
// an exhaustive list.
// See https://pkg.go.dev/gorm.io/gorm@v1.20.7#pkg-variables for list of
// errors.
func gormCode(err error) (codes.Code, bool) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return codes.NotFound, true
	}
	return codes.Unknown, false
}
