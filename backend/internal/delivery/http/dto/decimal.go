package dto

import (
	"bytes"
	"strconv"

	"github.com/shopspring/decimal"
)

type Decimal struct {
	value decimal.Decimal
	valid bool
}

func (d *Decimal) UnmarshalJSON(b []byte) error {
	raw := bytes.TrimSpace(b)
	if unquoted, err := strconv.Unquote(string(raw)); err == nil {
		raw = []byte(unquoted)
	}

	v, err := decimal.NewFromString(string(raw))
	d.value, d.valid = v, err == nil
	return nil
}
