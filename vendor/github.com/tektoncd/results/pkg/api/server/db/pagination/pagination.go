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

package pagination

import (
	"encoding/base64"
	"math"

	pb "github.com/tektoncd/results/pkg/api/server/db/pagination/proto/internal_go_proto"
	"google.golang.org/protobuf/proto"
)

// EncodeToken encodes a name + filter to an opaque page token
func EncodeToken(name, filter string) (token string, err error) {
	pi := &pb.ListPageIdentifier{
		Name:   name,
		Filter: filter,
	}
	var tokenByte []byte
	if tokenByte, err = proto.Marshal(pi); err != nil {
		return "", err
	}
	encodedResult := make([]byte, base64.RawURLEncoding.EncodedLen(len(tokenByte)))
	base64.RawURLEncoding.Encode(encodedResult, tokenByte)
	return base64.RawURLEncoding.EncodeToString(encodedResult), nil
}

// DecodeToken decodes an opaque page token into its name and filter parts.
func DecodeToken(token string) (name, filter string, err error) {
	var encodedToken []byte
	if encodedToken, err = base64.RawURLEncoding.DecodeString(token); err != nil {
		return "", "", err
	}
	tokenByte := make([]byte, base64.RawURLEncoding.DecodedLen(len(encodedToken)))
	if _, err = base64.RawURLEncoding.Decode(tokenByte, encodedToken); err != nil {
		return "", "", err
	}
	pi := &pb.ListPageIdentifier{}
	if err = proto.Unmarshal(tokenByte, pi); err != nil {
		return "", "", err
	}
	return pi.GetName(), pi.GetFilter(), err
}

// Batcher suggests dynamic batch sizes for list queries.
type Batcher struct {
	// Total number of items we want to fetch
	want int

	// Min/Max thresholds of number of items to fetch for a given batch.
	min, max int

	// ratio is used to detemine batch sizes relative to the wanted number of
	// results. This value changes each iteration based the number of items
	// successfully fetches and the total number of items fetched in the
	// previous batch. The less the previous ratio is, the bigger the upcoming
	// batch_size is.
	ratio float64
}

// NewBatcher creates a new batcher for the given requested page size.
func NewBatcher(want, min, max int) *Batcher {
	return &Batcher{
		want:  want,
		min:   min,
		max:   max,
		ratio: 1,
	}
}

// Update updates the Batcher based on the results of the last batch.
// `matched` is the number of items successfully matched by filters.
// `total` is the total number of rows last fetched.
// This calculates a new value to be used for calls to Next.
func (b *Batcher) Update(matched, total int) {
	b.ratio = float64(matched) / float64(total)
}

// Next returns the recommended next batch size to query.
func (b *Batcher) Next() int {
	n := int(math.Ceil(float64(b.want) / float64(b.ratio)))
	if n > b.max {
		return b.max
	}
	return n
}
