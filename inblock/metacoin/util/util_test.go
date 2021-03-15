package util

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func tParseNotNegative(t *testing.T, s string, isSuccess bool) {
	var d decimal.Decimal
	var e error

	d, e = ParseNotNegative(s)
	if isSuccess {
		if e != nil {
			t.Fatalf(`ParseNotNegative("%s") = %q, %v, Bad error occurred %#q`, s, d.String(), e, s)
		}
	} else {
		if e == nil {
			t.Fatalf(`ParseNotNegative("%s") = %q, Wrong success %#q`, s, d.String(), s)
		}
	}
}

func tParsePositive(t *testing.T, s string, isSuccess bool) {
	var d decimal.Decimal
	var e error

	d, e = ParsePositive(s)
	if isSuccess {
		if e != nil {
			t.Fatalf(` ParsePositive("%s") = %q, %v, Bad error occurred %#q`, s, d.String(), e, s)
		}
	} else {
		if e == nil {
			t.Fatalf(` ParsePositive("%s") = %q,  Wrong success %#q`, s, d.String(), s)
		}
	}
}

func tDataAssign(t *testing.T, s string, dataType string, minLength int, maxLength int, allowEmpty bool, isSuccess bool) {
	var dummy string
	var e error
	dummy = ""
	e = DataAssign(s, &dummy, dataType, minLength, maxLength, allowEmpty)
	if isSuccess {
		if e != nil {
			t.Fatalf(` DataAssign("%s") = %q, %v, Bad error occurred %#q`, s, dummy, e, s)
		}
		if allowEmpty == false {
			if dummy != strings.TrimSpace(s) {
				t.Fatalf(` DataAssign("%s") = %q, %v, not assign %#q`, s, dummy, e, s)
			}

		}
		t.Logf(` DataAssign("%s") = %q`, s, dummy)
	} else {
		if e == nil {
			t.Fatalf(` DataAssign("%s") = %q,  Wrong success %#q`, s, dummy, s)
		}
	}
}

func tNumericDataCheck(t *testing.T, s string, minValue string, maxValue string, maxDecimal int, allowEmpty bool, isSuccess bool) {
	var dummy string
	var e error
	e = NumericDataCheck(s, &dummy, minValue, maxValue, maxDecimal, allowEmpty)
	if isSuccess {
		if e != nil {
			t.Fatalf(` DataAssign("%s") = %q, %v, Bad error occurred %#q`, s, dummy, e, s)
		}
		if allowEmpty == false {
			if dummy != s {
				t.Fatalf(` DataAssign("%s") = %q, %v, not assign %#q`, s, dummy, e, s)
			}
		}
	} else {
		if e == nil {
			t.Fatalf(` DataAssign("%s") = %q,  Wrong success %#q`, s, dummy, s)
		}
	}
}

// TestHelloName calls greetings.Hello with a name, checking
// for a valid return value.
func TestParseNotNegative(t *testing.T) {
	// success check
	tParseNotNegative(t, "12345678901234567890123456789012345678901234567890123456789012345678901234567890", true) // long int
	tParseNotNegative(t, "123", true)                                                                              // normal int
	tParseNotNegative(t, "0", true)                                                                                // zero

	// fail check
	tParseNotNegative(t, "-123456789012345678901234567890123456789123456789012345678901234567890123456789", false)  // long negative
	tParseNotNegative(t, "12e3", false)                                                                             // exponential notation
	tParseNotNegative(t, "-1", false)                                                                               // negative
	tParseNotNegative(t, "0.000000000000000000000000000000000000000000000000000000000000000000000000000001", false) // long decimal
	tParseNotNegative(t, "0.1", false)                                                                              // decimal
	tParseNotNegative(t, "-1234.5", false)                                                                          // decimal + negative
	tParseNotNegative(t, "-12e3", false)                                                                            // negative exponential notation
	tParseNotNegative(t, "1234a", false)                                                                            // invalid
}

func TestParsePositive(t *testing.T) {
	// success check
	tParsePositive(t, "12345678901234567890123456789012345678901234567890123456789012345678901234567890", true)
	tParsePositive(t, "123", true)

	// fail check
	tParsePositive(t, "0", false)
	tParsePositive(t, "-123456789012345678901234567890123456789123456789012345678901234567890123456789", false)  // long negative
	tParsePositive(t, "12e3", false)                                                                             // exponential notation
	tParsePositive(t, "-1", false)                                                                               // negative
	tParsePositive(t, "0.000000000000000000000000000000000000000000000000000000000000000000000000000001", false) // long decimal
	tParsePositive(t, "0.1", false)                                                                              // decimal
	tParsePositive(t, "-1234.5", false)                                                                          // decimal + negative
	tParsePositive(t, "-12e3", false)                                                                            // negative exponential notation
	tParsePositive(t, "1234a", false)                                                                            // invalid
}

func TestDataAssign(t *testing.T) {
	// success check
	tDataAssign(t, "MTiiiiiiiiiiiiMETACOINiiiiiiiiii79c3496e", "address", 10, 20, false, true) // address ignore min/max length
	tDataAssign(t, "https://domain.com", "url", 3, 40, false, true)
	tDataAssign(t, "https://domain.com/path", "url", 3, 40, false, true)
	tDataAssign(t, "https://domain.com/path?q=3", "url", 3, 40, false, true)
	tDataAssign(t, "https://domain.com/path?q={id}", "url", 3, 40, false, true)
	tDataAssign(t, "https://domain.com/path1/path2/path3/?q={id}", "url", 3, 90, false, true)
	tDataAssign(t, "https://domain.com:80/path1/path2/path3/?q={id}", "url", 3, 90, false, true)
	tDataAssign(t, "https://userid:userpw@domain.com:80/path1/path2/path3/?q={id}", "url", 3, 90, false, true)
	tDataAssign(t, "https://domain.com:80", "url", 3, 90, false, true)
	tDataAssign(t, "https://domain.com:80/    ", "url", 3, 90, false, true)
	tDataAssign(t, "https:/domain.com", "url", 3, 40, false, true)
	tDataAssign(t, "http://domain.com/path1/path2/path3/?q={id}", "url", 3, 90, false, true)
	tDataAssign(t, "    http://192.168.1.5/path1/path2/path3/?q={id}", "url", 3, 90, false, true)
	tDataAssign(t, "http://192.168.1.5:80/path1/path2/path3/?q={id}", "url", 3, 90, false, true)
	tDataAssign(t, "abcdefghijklmnopqrstuvwxyz", "id", 3, 90, false, true)
	tDataAssign(t, "abcdefghijklmnopqrstuvwxyz", "id", 26, 26, false, true)
	tDataAssign(t, "for test string 1234 {},.", "string", 3, 90, false, true)

	tDataAssign(t, "", "string", 3, 90, true, true)
	tDataAssign(t, "", "address", 3, 90, true, true)
	tDataAssign(t, "", "url", 3, 90, true, true)
	tDataAssign(t, "", "id", 3, 90, true, true)

	// fail check
	tDataAssign(t, "MTiiiiiiiiiiiiMETACOINiiiiiiiiii79c3496a", "address", 10, 20, false, false)
	tDataAssign(t, "httpx://domain.com/path", "url", 3, 40, false, false)
	tDataAssign(t, "domain.com/path", "url", 3, 40, false, false)
	tDataAssign(t, "abcdefghijklmnopqrstuvwxyz", "id", 10, 10, false, false)
	tDataAssign(t, "for test string 1234 {},.", "string", 60, 90, false, false)
	tDataAssign(t, "for test string 1234 {},.", "string", 20, 21, false, false)

}

func TestNumericDataCheck(t *testing.T) {
	tNumericDataCheck(t, "123.1", "100", "200", 1, false, true)
	tNumericDataCheck(t, "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890.1", "100", "9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999", 1, false, true)
	tNumericDataCheck(t, "1.1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890", "1", "99999999999999999999999999999999999999999999999999", 999, false, true)
	tNumericDataCheck(t, "123.1", "100", "200", 5, false, true)
	tNumericDataCheck(t, "123", "100", "200", 3, false, true)
	tNumericDataCheck(t, "123", "100", "200", 0, false, true)
	tNumericDataCheck(t, "-12", "-100", "200", 0, false, true)

	// fail check
	tNumericDataCheck(t, "123", "10", "100", 0, false, false)
	tNumericDataCheck(t, "12a", "100", "200", 0, false, false)
	tNumericDataCheck(t, "1 2", "100", "200", 0, false, false)
	tNumericDataCheck(t, "-12", "100", "200", 0, false, false)
	tNumericDataCheck(t, "1-2", "-100", "200", 0, false, false)
	tNumericDataCheck(t, "123", "1000", "10000", 0, false, false)
	tNumericDataCheck(t, "123.123", "1000", "10000", 2, false, false)
}
