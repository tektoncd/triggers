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
	"fmt"

	"github.com/tektoncd/results/pkg/api/server/db/pagination"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	minPageSize = 50
	maxPageSize = 10000
)

func pageSize(in int) (int, error) {
	if in < 0 {
		return 0, status.Error(codes.InvalidArgument, "PageSize should be greater than 0")
	} else if in == 0 {
		return minPageSize, nil
	} else if in > maxPageSize {
		return maxPageSize, nil
	}
	return in, nil
}

func pageStart(token, filter string) (string, error) {
	if token == "" {
		return "", nil
	}

	tokenName, tokenFilter, err := pagination.DecodeToken(token)
	if err != nil {
		return "", status.Error(codes.InvalidArgument, fmt.Sprintf("invalid PageToken: %v", err))
	}
	if filter != tokenFilter {
		return "", status.Error(codes.InvalidArgument, "filter does not match previous query")
	}
	return tokenName, nil
}
