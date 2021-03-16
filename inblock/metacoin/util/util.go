package util

import (
	"crypto/rand"
	"errors"
	"fmt"
	"hash"
	"math/big"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"

	"crypto/ecdsa"
	"crypto/sha512"
	"crypto/sha256"
	"crypto/x509"
    "crypto/md5"
	"hash/crc32"

	"github.com/shopspring/decimal"
)

// ecdsaSignature : ecdsa signature
type ecdsaSignature struct {
	R, S *big.Int
}

// MakeRandomString : make random string!
func MakeRandomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

func GetMD5(data string) string {
	h := md5.New()
	return fmt.Sprintf("%x", h.Sum([]byte(data)))
}

func GenerateKey(prefix string, data []string) string {
	h := md5.New()
	return fmt.Sprintf("%6s_%x", h.Sum([]byte(strings.Join(data, "|"))))
}


// EcdsaSignVerify : ecdsa signature verify
func EcdsaSignVerify(PublicKeyPem, Data, Sign string) error {
	var data = []byte(Data)
	var signature []byte
	var pub interface{}
	var pubkey *ecdsa.PublicKey
	var ok bool
	var err error
	var esig ecdsaSignature
	var block *pem.Block
	var hasher hash.Hash

	signature, _ = base64.StdEncoding.DecodeString(Sign)

	if len(PublicKeyPem) < 60 {
		return errors.New("2310,Invalid Public key")
	}
	block, _ = pem.Decode([]byte(PublicKeyPem))
	if block == nil {
		if strings.Index(PublicKeyPem, "\n") == -1 {
			var dt = len(PublicKeyPem) - 24
			var buf = make([]string, 3)
			buf[0] = PublicKeyPem[0:26]
			buf[1] = PublicKeyPem[26:dt]
			buf[2] = PublicKeyPem[dt:]
			PublicKeyPem = strings.Join(buf, "\n")
		}
		block, _ = pem.Decode([]byte(PublicKeyPem))
		if block == nil {
			return errors.New("2310,Public key decode error")
		}
	}

	pub, err = x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}

	switch pub.(type) {
	case *ecdsa.PublicKey:
		break
	default:
		return errors.New("2210,PublicKey type error")
	}

	pubkey, ok = pub.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("2110,PublicKey format error")
	}

	if _, err = asn1.Unmarshal(signature, &esig); err != nil {
		return errors.New("2230,Signature format error - " + err.Error())
	}

	hasher = sha512.New384()	// for OPENSSL_ALGO_SHA384
	hasher.Write([]byte(data))
	if ecdsa.Verify(pubkey, hasher.Sum(nil), esig.R, esig.S) {
		return nil
	}

	hasher = sha512.New()	// for OPENSSL_ALGO_SHA512
	hasher.Write([]byte(data))
	if ecdsa.Verify(pubkey, hasher.Sum(nil), esig.R, esig.S) {
		return nil
	}

	hasher = sha256.New()  // for OPENSSL_ALGO_SHA256
	hasher.Write([]byte(data))
	if ecdsa.Verify(pubkey, hasher.Sum(nil), esig.R, esig.S) {
		return nil
	}

		return errors.New("2010,Invalid signature")
	}

// DataAssign string length check and return trimming
func DataAssign(src string, dest *string, dataType string, minLength int, maxLength int, allowEmpty bool) error {
	var err error
	var u *url.URL
	buf := strings.TrimSpace(src)
	if allowEmpty == true && len(buf) == 0 {
		return nil
	}

	if dataType == "address" {
		if len(strings.TrimSpace(buf)) != 40 {
			return errors.New("not address")
		}
		if buf[:2] != "MT" {
			return errors.New("not address")
		}

		calcCRC := fmt.Sprintf("%08x", crc32.Checksum([]byte(buf[2:32]), crc32.MakeTable(crc32.IEEE)))
		if buf[32:] != calcCRC {
			return errors.New("not address")
		}
	} else if dataType == "url" {
		if u, err = url.ParseRequestURI(buf); err != nil {
			return errors.New("invalid url")
		}

		if u.Scheme != "http" && u.Scheme != "https" {
			return errors.New("scheme is must http or https")
		}
		if len(buf) > maxLength {
			return errors.New("too long")
		}
	} else if dataType == "id" {
		r, _ := regexp.Compile("^[a-zA-Z0-9]{" + strconv.Itoa(minLength) + "," + strconv.Itoa(maxLength) + "}$")
		if r.MatchString(buf) == false {
			return errors.New("not valid data")
		}
	} else {
		if len(buf) < minLength {
			return errors.New("too short")
		}

		if len(buf) > maxLength {
			return errors.New("too long")
		}
	}

	*dest = buf
	return nil
}

// NumericDataCheck string length check and return trimming
func NumericDataCheck(src string, dest *string, minValue string, maxValue string, maxDecimal int, allowEmpty bool) error {
	var err error
	var dmin, dmax, dsrc decimal.Decimal

	buf := strings.TrimSpace(src)
	if allowEmpty == true && len(buf) == 0 {
		return nil
	}

	if maxDecimal > 0 {
		r, _ := regexp.Compile("^[-]?[0-9]+(|(\\.[0-9]{1," + strconv.Itoa(maxDecimal) + "}))$")
		if r.MatchString(buf) == false {
			return errors.New(" is invalid data")
		}
	} else {
		r, _ := regexp.Compile("^[-]?[0-9]+$")
		if r.MatchString(buf) == false {
			return errors.New(" is invalid data")
		}
	}

	if dmin, err = decimal.NewFromString(minValue); err != nil {
		return errors.New(" minValue invalid")
	}

	if dmax, err = decimal.NewFromString(maxValue); err != nil {
		return errors.New(" maxValue invalid")
	}

	if dsrc, err = decimal.NewFromString(buf); err != nil {
		return err
	}

	if dsrc.Cmp(dmin) < 0 {
		return errors.New(" must be bigger then " + minValue)
	}

	if dsrc.Cmp(dmax) > 0 {
		return errors.New(" must be smaller then " + maxValue)
	}

	*dest = buf
	return nil
}

// IsAddress - address check
func IsAddress(address string) bool {
	if len(strings.TrimSpace(address)) != 40 {
		return true
	}
	if address[:2] != "MT" {
		return true
	}

	calcCRC := fmt.Sprintf("%08x", crc32.Checksum([]byte(address[2:32]), crc32.MakeTable(crc32.IEEE)))
	if address[32:] != calcCRC {
		return true
	}
	return false
}

// Strtoint to int.
func Strtoint(data string) (int, error) {
	var err error
	var i int
	if i, err = strconv.Atoi(data); err != nil {
		return 0, errors.New("9900,Data [" + data + "] is not integer")
	}
	return i, nil
}

// Strtoint64 to int.
func Strtoint64(data string) (int64, error) {
	var err error
	var i64 int64
	if i64, err = strconv.ParseInt(data, 10, 64); err != nil {
		return 0, errors.New("9900,Data [" + data + "] is not integer")
	}
	return i64, nil
}

// GetOrdNumber number to ord
func GetOrdNumber(idx int) string {
	if idx == 1 {
		return strconv.Itoa(idx) + "st"
	} else if idx == 2 {
		return strconv.Itoa(idx) + "nd"
	} else if idx == 3 {
		return strconv.Itoa(idx) + "rd"
	} else {
		return strconv.Itoa(idx) + "th"
	}
}

// RemoveElement slice element
func RemoveElement(s []string, i int) []string {
	if i < 0 {
		return s
	}
	if i >= len(s) {
		i = len(s) - 1
	}

	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func isNumeric(s string) bool {
	for _, v := range s {
		if v < '0' || v > '9' {
			return false
		}
	}
	return true
}

// ParsePositive string is positive ?
func ParsePositive(s string) (decimal.Decimal, error) {
	var d decimal.Decimal
	var err error

	if d, err = decimal.NewFromString(s); err != nil {
		return d, errors.New("1101, " + s + " is not integer string")
	}
	if isNumeric(s) == false {
		return d, errors.New("1101, " + s + " is not integer string")
	}
	if !d.IsPositive() {
		return d, errors.New("1101, " + s + " is either 0 or negative.")
	}
	return d, nil
}

// ParseNotNegative string is positive or zero ?
func ParseNotNegative(s string) (decimal.Decimal, error) {
	var d decimal.Decimal
	var err error

	if d, err = decimal.NewFromString(s); err != nil {
		return d, errors.New("1101, " + s + " is not integer string")
	}

	if isNumeric(s) == false {
		return d, errors.New("1101, " + s + " is not integer string")
	}

	if d.IsNegative() {
		return d, errors.New("1101," + s + " is negative.")
	}
	return d, nil
}
