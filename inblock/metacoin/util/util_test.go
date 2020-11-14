package util

import (
	"testing"

	"github.com/shopspring/decimal"
)

func tParseNotNegative(t *testing.T, s string, isError bool) {
	var d decimal.Decimal
	var e error

	d, e = ParseNotNegative(s)
	if isError {
		if e == nil {
			t.Fatalf(`TestParseNotNegative("%s") = %q, %v, want match for %#q`, s, d.String(), e, s)
		}
	} else {
		if e != nil {
			t.Fatalf(`TestParseNotNegative("%s") = %q, %v, want match for %#q`, s, d.String(), e, s)
		}
	}
}

func tParsePositive(t *testing.T, s string, isError bool) {
	var d decimal.Decimal
	var e error

	d, e = ParsePositive(s)
	if isError {
		if e == nil {
			t.Fatalf(`ParsePositive("%s") = %q, %v, want match for %#q`, s, d.String(), e, s)
		}
	} else {
		if e != nil {
			t.Fatalf(`ParsePositive("%s") = %q, %v, want match for %#q`, s, d.String(), e, s)
		}
	}
}

// TestHelloName calls greetings.Hello with a name, checking
// for a valid return value.
func TestParseNotNegative(t *testing.T) {

	tParseNotNegative(t, "12345678890", false)
	tParseNotNegative(t, "0", false)
	tParseNotNegative(t, "1234.5", true)
	tParseNotNegative(t, "-1", true)
	tParseNotNegative(t, "-0.00000000000000001", true)
	tParseNotNegative(t, "12e3", true)
}

func TestParsePositive(t *testing.T) {
	tParsePositive(t, "0", true)
	tParsePositive(t, "12345678890", false)
	tParsePositive(t, "1234.5", true)
	tParsePositive(t, "-1", true)
	tParsePositive(t, "-0.00000000000000001", true)
	tParsePositive(t, "12e3", true)
}
