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

	switch v := v.(type) {
	case float64:
		*d = Duration(time.Duration(v))
		return nil
	case string:
		dur, err := time.ParseDuration(v)
		if err != nil {
			return err
		}
		*d = Duration(dur)
		return nil
	}

	return errors.New("invalid duration")
}
