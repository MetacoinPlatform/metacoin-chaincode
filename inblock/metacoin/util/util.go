package util

import (
	"crypto/rand"
	"errors"
	"fmt"
	"hash"
	"io"
	"math/big"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"

	"crypto/ecdsa"
	"crypto/md5"
	"crypto/sha256"
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

func GetMD5(data string) string {
	h := md5.New()
	io.WriteString(h, data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func GenerateKey(prefix string, data []string) string {
	prefix = prefix + "000000"
	return fmt.Sprintf("%6s_%s", prefix[:6], GetMD5(strings.Join(data, "|")))
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
		if !strings.Contains(PublicKeyPem, "\n") {
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

	hasher = sha512.New384() // for OPENSSL_ALGO_SHA384
	hasher.Write([]byte(data))
	if ecdsa.Verify(pubkey, hasher.Sum(nil), esig.R, esig.S) {
		return nil
	}

	hasher = sha512.New() // for OPENSSL_ALGO_SHA512
	hasher.Write([]byte(data))
	if ecdsa.Verify(pubkey, hasher.Sum(nil), esig.R, esig.S) {
		return nil
	}

	hasher = sha256.New() // for OPENSSL_ALGO_SHA256
	hasher.Write([]byte(data))
	if ecdsa.Verify(pubkey, hasher.Sum(nil), esig.R, esig.S) {
		return nil
	}

	return errors.New("2010,Invalid signature")
}

// DataAssign string length check and return trimming
/*
	dataType : address, url, string, bool, id
	minLength : minimum length, if 0 then allow data clear
	allowEmpty : if true and minlength > 0 and buf is empty string then not update dest.
*/
func DataAssign(src string, dest *string, dataType string, minLength int, maxLength int, allowEmpty bool) error {
	var err error
	var u *url.URL
	buf := strings.TrimSpace(src)

	// input data is null string.
	if len(buf) == 0 {
		// allow clear
		if minLength == 0 {
			*dest = buf
			return nil
		}
		// allow empay for not update
		if allowEmpty {
			return nil
		}
	}

	if dataType == "address" {
		if len(strings.TrimSpace(buf)) != 40 {
			return errors.New("Invalid METACOIN address - " + buf)
		}
		if buf[:2] != "MT" {
			return errors.New("Invalid METACOIN address - " + buf)
		}

		calcCRC := fmt.Sprintf("%08x", crc32.Checksum([]byte(buf[2:32]), crc32.MakeTable(crc32.IEEE)))
		if buf[32:] != calcCRC {
			return errors.New("Invalid METACOIN address - " + buf)
		}
	} else if dataType == "url" {
		if u, err = url.ParseRequestURI(buf); err != nil {
			return errors.New("invalid url - " + buf)
		}

		if u.Scheme != "http" && u.Scheme != "https" {
			return errors.New("scheme is must http or https - " + buf)
		}
		if len(buf) > maxLength {
			return errors.New("too long")
		}
	} else if dataType == "bool" {
		if buf != "1" && buf == "0" {
			return errors.New("data is must be 1 or 0 - " + buf)
		}
	} else if dataType == "id" {
		r, _ := regexp.Compile("^[a-zA-Z0-9]{" + strconv.Itoa(minLength) + "," + strconv.Itoa(maxLength) + "}$")
		if !r.MatchString(buf) {
			return errors.New("not valid data - " + buf)
		}
	} else {
		if len(buf) < minLength {
			return errors.New("too short")
		}

		if len(buf) > maxLength {
			return errors.New("too long")
		}
	}

	if dest != nil {
		*dest = buf
	}
	return nil
}

func DecimalCountCheck(src string, maxDecimal int) error {
	if maxDecimal == 0 {
		if strings.Contains(src, ".") {
			return errors.New("decimal is big")
		}
	}
	if maxDecimal > 0 {
		r, _ := regexp.Compile("^[-]?[0-9]+(|(\\.[0-9]{1," + strconv.Itoa(maxDecimal) + "}))$")
		if !r.MatchString(src) {
			return errors.New(" is invalid data")
		}
	} else {
		r, _ := regexp.Compile("^[-]?[0-9]+$")
		if !r.MatchString(src) {
			return errors.New(" is invalid data")
		}
	}

	return nil
}

func MulDecimal(dsrc decimal.Decimal, floatSize int32) decimal.Decimal {
	var cRate, mu decimal.Decimal
	cRate, _ = decimal.NewFromString("10")
	mu = decimal.NewFromInt32(floatSize)
	return dsrc.Mul(cRate.Pow(mu))
}

func DivDecimal(dsrc decimal.Decimal, floatSize int32) decimal.Decimal {
	var cRate, mu decimal.Decimal
	cRate, _ = decimal.NewFromString("10")
	mu = decimal.NewFromInt32(floatSize)
	return dsrc.Div(cRate.Pow(mu)).Floor()
}

// NumericDataCheck string length check and return trimming
func NumericDataCheck(src string, dest *string, minValue string, maxValue string, maxDecimal int, allowEmpty bool) error {
	var err error
	var dmin, dmax, dsrc decimal.Decimal

	buf := strings.TrimSpace(src)
	if allowEmpty && len(buf) == 0 {
		return nil
	}

	if maxDecimal > 0 {
		r, _ := regexp.Compile("^[-]?[0-9]+(|(\\.[0-9]{1," + strconv.Itoa(maxDecimal) + "}))$")
		if !r.MatchString(buf) {
			return errors.New(" is invalid data")
		}
	} else {
		r, _ := regexp.Compile("^[-]?[0-9]+$")
		if !r.MatchString(buf) {
			return errors.New(" is invalid data")
		}
	}

	if dsrc, err = decimal.NewFromString(buf); err != nil {
		return err
	}

	if minValue != "" {
		if dmin, err = decimal.NewFromString(minValue); err != nil {
			return errors.New(" minValue invalid")
		}

		if dsrc.Cmp(dmin) < 0 {
			return errors.New(" must be bigger then " + minValue)
		}
	}

	if maxValue != "" {
		if dmax, err = decimal.NewFromString(maxValue); err != nil {
			return errors.New(" maxValue invalid")
		}

		if dsrc.Cmp(dmax) > 0 {
			return errors.New(" must be smaller then " + maxValue)
		}
	}

	if dest != nil {
		*dest = dsrc.String()
	}
	return nil
}

// IsAddress - address check
func IsAddress(address string) bool {
	if len(strings.TrimSpace(address)) != 40 {
		return false
	}
	if address[:2] != "MT" {
		return false
	}

	calcCRC := fmt.Sprintf("%08x", crc32.Checksum([]byte(address[2:32]), crc32.MakeTable(crc32.IEEE)))
	return address[32:] == calcCRC
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

	if !isNumeric(s) {
		return d, errors.New("1101, " + s + " is not integer string")
	}

	if d, err = decimal.NewFromString(s); err != nil {
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

	if !isNumeric(s) {
		return d, errors.New("1101, " + s + " is not integer string")
	}

	if d, err = decimal.NewFromString(s); err != nil {
		return d, errors.New("1101, " + s + " is not integer string")
	}

	if d.IsNegative() {
		return d, errors.New("1101," + s + " is negative.")
	}

	return d, nil
}

// JSONEncode simple
func JSONEncode(v interface{}) string {
	if argdat, err := json.Marshal(v); err == nil {
		return string(argdat)
	} else {
		return ""
	}
}

func ISO3166Check(country_code string) error {
	if country_code == "" {
		return nil
	}
	var iso3166 = [...]string{
		"AD", // Andorra
		"AE", // United Arab Emirates
		"AF", // Afghanistan
		"AG", // Antigua and Barbuda
		"AI", // Anguilla
		"AL", // Albania
		"AM", // Armenia
		"AO", // Angola
		"AQ", // Antarctica
		"AR", // Argentina
		"AS", // American Samoa
		"AT", // Austria
		"AU", // Australia
		"AW", // Aruba
		"AX", // Åland Islands
		"AZ", // Azerbaijan
		"BA", // Bosnia and Herzegovina
		"BB", // Barbados
		"BD", // Bangladesh
		"BE", // Belgium
		"BF", // Burkina Faso
		"BG", // Bulgaria
		"BH", // Bahrain
		"BI", // Burundi
		"BJ", // Benin
		"BL", // Saint Barthélemy
		"BM", // Bermuda
		"BN", // Brunei Darussalam
		"BO", // Bolivia (Plurinational State of)
		"BQ", // Bonaire, Sint Eustatius and Saba
		"BR", // Brazil
		"BS", // Bahamas
		"BT", // Bhutan
		"BV", // Bouvet Island
		"BW", // Botswana
		"BY", // Belarus
		"BZ", // Belize
		"CA", // Canada
		"CC", // Cocos (Keeling) Islands
		"CD", // Congo (Democratic Republic of the)
		"CF", // Central African Republic
		"CG", // Congo
		"CH", // Switzerland
		"CI", // Côte d'Ivoire
		"CK", // Cook Islands
		"CL", // Chile
		"CM", // Cameroon
		"CN", // China
		"CO", // Colombia
		"CR", // Costa Rica
		"CU", // Cuba
		"CV", // Cabo Verde
		"CW", // Curaçao
		"CX", // Christmas Island
		"CY", // Cyprus
		"CZ", // Czechia
		"DE", // Germany
		"DJ", // Djibouti
		"DK", // Denmark
		"DM", // Dominica
		"DO", // Dominican Republic
		"DZ", // Algeria
		"EC", // Ecuador
		"EE", // Estonia
		"EG", // Egypt
		"EH", // Western Sahara
		"ER", // Eritrea
		"ES", // Spain
		"ET", // Ethiopia
		"FI", // Finland
		"FJ", // Fiji
		"FK", // Falkland Islands (Malvinas)
		"FM", // Micronesia (Federated States of)
		"FO", // Faroe Islands
		"FR", // France
		"GA", // Gabon
		"GB", // United Kingdom of Great Britain and Northern Ireland
		"GD", // Grenada
		"GE", // Georgia
		"GF", // French Guiana
		"GG", // Guernsey
		"GH", // Ghana
		"GI", // Gibraltar
		"GL", // Greenland
		"GM", // Gambia
		"GN", // Guinea
		"GP", // Guadeloupe
		"GQ", // Equatorial Guinea
		"GR", // Greece
		"GS", // South Georgia and the South Sandwich Islands
		"GT", // Guatemala
		"GU", // Guam
		"GW", // Guinea-Bissau
		"GY", // Guyana
		"HK", // Hong Kong
		"HM", // Heard Island and McDonald Islands
		"HN", // Honduras
		"HR", // Croatia
		"HT", // Haiti
		"HU", // Hungary
		"ID", // Indonesia
		"IE", // Ireland
		"IL", // Israel
		"IM", // Isle of Man
		"IN", // India
		"IO", // British Indian Ocean Territory
		"IQ", // Iraq
		"IR", // Iran (Islamic Republic of)
		"IS", // Iceland
		"IT", // Italy
		"JE", // Jersey
		"JM", // Jamaica
		"JO", // Jordan
		"JP", // Japan
		"KE", // Kenya
		"KG", // Kyrgyzstan
		"KH", // Cambodia
		"KI", // Kiribati
		"KM", // Comoros
		"KN", // Saint Kitts and Nevis
		"KP", // Korea (Democratic People's Republic of)
		"KR", // Korea (Republic of)
		"KW", // Kuwait
		"KY", // Cayman Islands
		"KZ", // Kazakhstan
		"LA", // Lao People's Democratic Republic
		"LB", // Lebanon
		"LC", // Saint Lucia
		"LI", // Liechtenstein
		"LK", // Sri Lanka
		"LR", // Liberia
		"LS", // Lesotho
		"LT", // Lithuania
		"LU", // Luxembourg
		"LV", // Latvia
		"LY", // Libya
		"MA", // Morocco
		"MC", // Monaco
		"MD", // Moldova (Republic of)
		"ME", // Montenegro
		"MF", // Saint Martin (French part)
		"MG", // Madagascar
		"MH", // Marshall Islands
		"MK", // North Macedonia
		"ML", // Mali
		"MM", // Myanmar
		"MN", // Mongolia
		"MO", // Macao
		"MP", // Northern Mariana Islands
		"MQ", // Martinique
		"MR", // Mauritania
		"MS", // Montserrat
		"MT", // Malta
		"MU", // Mauritius
		"MV", // Maldives
		"MW", // Malawi
		"MX", // Mexico
		"MY", // Malaysia
		"MZ", // Mozambique
		"NA", // Namibia
		"NC", // New Caledonia
		"NE", // Niger
		"NF", // Norfolk Island
		"NG", // Nigeria
		"NI", // Nicaragua
		"NL", // Netherlands
		"NO", // Norway
		"NP", // Nepal
		"NR", // Nauru
		"NU", // Niue
		"NZ", // New Zealand
		"OM", // Oman
		"PA", // Panama
		"PE", // Peru
		"PF", // French Polynesia
		"PG", // Papua New Guinea
		"PH", // Philippines
		"PK", // Pakistan
		"PL", // Poland
		"PM", // Saint Pierre and Miquelon
		"PN", // Pitcairn
		"PR", // Puerto Rico
		"PS", // Palestine, State of
		"PT", // Portugal
		"PW", // Palau
		"PY", // Paraguay
		"QA", // Qatar
		"RE", // Réunion
		"RO", // Romania
		"RS", // Serbia
		"RU", // Russian Federation
		"RW", // Rwanda
		"SA", // Saudi Arabia
		"SB", // Solomon Islands
		"SC", // Seychelles
		"SD", // Sudan
		"SE", // Sweden
		"SG", // Singapore
		"SH", // Saint Helena, Ascension and Tristan da Cunha
		"SI", // Slovenia
		"SJ", // Svalbard and Jan Mayen
		"SK", // Slovakia
		"SL", // Sierra Leone
		"SM", // San Marino
		"SN", // Senegal
		"SO", // Somalia
		"SR", // Suriname
		"SS", // South Sudan
		"ST", // Sao Tome and Principe
		"SV", // El Salvador
		"SX", // Sint Maarten (Dutch part)
		"SY", // Syrian Arab Republic
		"SZ", // Eswatini
		"TC", // Turks and Caicos Islands
		"TD", // Chad
		"TF", // French Southern Territories
		"TG", // Togo
		"TH", // Thailand
		"TJ", // Tajikistan
		"TK", // Tokelau
		"TL", // Timor-Leste
		"TM", // Turkmenistan
		"TN", // Tunisia
		"TO", // Tonga
		"TR", // Turkey
		"TT", // Trinidad and Tobago
		"TV", // Tuvalu
		"TW", // Taiwan (Province of China)
		"TZ", // Tanzania, United Republic of
		"UA", // Ukraine
		"UG", // Uganda
		"UM", // United States Minor Outlying Islands
		"US", // United States of America
		"UY", // Uruguay
		"UZ", // Uzbekistan
		"VA", // Holy See
		"VC", // Saint Vincent and the Grenadines
		"VE", // Venezuela (Bolivarian Republic of)
		"VG", // Virgin Islands (British)
		"VI", // Virgin Islands (U.S.)
		"VN", // Viet Nam
		"VU", // Vanuatu
		"WF", // Wallis and Futuna
		"WS", // Samoa
		"YE", // Yemen
		"YT", // Mayotte
		"ZA", // South Africa
		"ZM", // Zambia
		"ZW", // Zimbabwe
	}

	for _, n := range iso3166 {
		if country_code == n {
			return nil
		}
	}
	return errors.New(country_code + " is not country code")
}
