package util

import (
	"crypto/rand"
	"errors"
	"fmt"
	"hash"
	"math/big"
	"strconv"
	"strings"

	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"

	"crypto/ecdsa"
	"crypto/sha512"
	"crypto/x509"
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
			buf[2] = PublicKeyPem[dt:len(PublicKeyPem)]
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

	switch pubkey.Curve.Params().BitSize {
	case 384:
		hasher = sha512.New384()
		break
	case 521:
		hasher = sha512.New()
		break
	default:
		return errors.New("2220,Invalid public key curve")
	}

	if _, err = asn1.Unmarshal(signature, &esig); err != nil {
		return errors.New("2230,Signature format error - " + err.Error())
	}

	hasher.Write([]byte(data))
	if ecdsa.Verify(pubkey, hasher.Sum(nil), esig.R, esig.S) {
		return nil
	} else {
		return errors.New("2010,Invalid signature")
	}
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
