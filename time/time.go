// SPDX-License-Identifier: BSD-3-Clause

// Package time provides a duration type that is serialized as a human-readable
// string instead of a number of nanoseconds.
package time

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration is a [time.Duration] that is marshaled to JSON and YAML as a
// string, e.g. "1m30s".
//
// It is a defined type, not an alias, so the methods of [time.Duration] are
// not available on it: use a conversion to get them back.
type Duration time.Duration

var (
	_ json.Marshaler   = (*Duration)(nil)
	_ json.Unmarshaler = (*Duration)(nil)
)

// MarshalJSON implements [json.Marshaler].
//
// The duration is written as a string, e.g. "1m30s".
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON implements [json.Unmarshaler].
//
// Both a duration string ("1m30s") and a number of nanoseconds are accepted.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	d2, err := fromRaw(v)
	if err != nil {
		return err
	}

	*d = d2
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface.
//
// The duration is written as a string, e.g. "1m30s".
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
//
// Both a duration string ("1m30s") and a number of nanoseconds are accepted.
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}
	if err := unmarshal(&v); err != nil {
		return err
	}

	d2, err := fromRaw(v)
	if err != nil {
		return err
	}

	*d = d2
	return nil
}

// fromRaw converts a decoded JSON or YAML value into a Duration.
//
// A string is parsed as a duration, a number is treated as a number of
// nanoseconds. Any other value is rejected.
//
// The JSON decoder produces the numbers as float64, while the YAML decoder
// produces the whole numbers as int and the fractional ones as float64.
func fromRaw(v interface{}) (Duration, error) {
	switch v := v.(type) {
	case float64:
		return Duration(time.Duration(v)), nil
	case int:
		return Duration(time.Duration(v)), nil
	case string:
		dur, err := time.ParseDuration(v)
		if err != nil {
			return Duration(0), err
		}
		return Duration(dur), nil
	default:
		return Duration(0), fmt.Errorf("invalid duration: %v (%T)", v, v)
	}
}
