// SPDX-License-Identifier: BSD-3-Clause

package protowrap

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Ptr constructs a pointer to the value of any type.
func Ptr[T any](v T) *T {
	return &v
}

// FromTimePtr constructs an instance of timestamppb.Timestamp from the provided time.Time.
// If the nil has passed, then the nil will be returned.
func FromTimePtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
