// SPDX-License-Identifier: BSD-3-Clause

package time

import (
	"encoding/json"
	"errors"
	"time"
)

type Duration time.Duration

var (
	_ json.Marshaler   = (*Duration)(nil)
	_ json.Unmarshaler = (*Duration)(nil)
)

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

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

func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

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

func fromRaw(v interface{}) (Duration, error) {
	switch v := v.(type) {
	case float64:
		return Duration(time.Duration(v)), nil
	case string:
		dur, err := time.ParseDuration(v)
		if err != nil {
			return Duration(0), err
		}
		return Duration(dur), nil
	default:
		return Duration(0), errors.New("invalid duration")
	}
}
