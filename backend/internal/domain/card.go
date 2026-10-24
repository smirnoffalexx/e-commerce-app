package domain

import (
	"strings"
	"time"
)

const (
	minCardDigits     = 12
	maxCardDigits     = 19
	maxCardYearsAhead = 20
)

type Card struct {
	Number   string
	ExpMonth int
	ExpYear  int
	CVC      string
}

func NormalizeCardNumber(number string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' {
			return -1
		}
		return r
	}, strings.TrimSpace(number))
}

func (c Card) Last4() string {
	n := NormalizeCardNumber(c.Number)
	if len(n) < 4 {
		return n
	}
	return n[len(n)-4:]
}

func (c Card) validate(verr *ValidationError, now time.Time) {
	number := NormalizeCardNumber(c.Number)
	switch {
	case number == "":
		verr.Add("payment.card_number", CodeRequired, "card number is required")
	case !isDigits(number) || len(number) < minCardDigits || len(number) > maxCardDigits:
		verr.Add("payment.card_number", CodeInvalidFormat, "card number must contain 12 to 19 digits")
	case !luhnValid(number):
		verr.Add("payment.card_number", CodeInvalidFormat, "card number is invalid")
	}

	now = now.UTC()
	monthOK := c.ExpMonth >= 1 && c.ExpMonth <= 12
	yearOK := c.ExpYear >= now.Year() && c.ExpYear <= now.Year()+maxCardYearsAhead
	if !monthOK {
		verr.Add("payment.exp_month", CodeInvalidFormat, "expiry month must be between 1 and 12")
	}
	switch {
	case !yearOK:
		verr.Add("payment.exp_year", CodeInvalidFormat, "expiry year must be a four-digit year, not in the past")
	case monthOK && c.ExpYear == now.Year() && c.ExpMonth < int(now.Month()):
		verr.Add("payment.exp_month", "expired", "card has expired")
	}

	cvc := strings.TrimSpace(c.CVC)
	if len(cvc) < 3 || len(cvc) > 4 || !isDigits(cvc) {
		verr.Add("payment.cvc", CodeInvalidFormat, "cvc must be 3 or 4 digits")
	}
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

func luhnValid(number string) bool {
	sum := 0
	double := false
	for i := len(number) - 1; i >= 0; i-- {
		d := int(number[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}
