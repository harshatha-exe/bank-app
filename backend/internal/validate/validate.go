package validate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"bank-app-ioc/backend/internal/apperr"
)

// ---------- collecting field errors ----------

type Validator struct{ fields map[string]string }

func New() *Validator { return &Validator{fields: map[string]string{}} }

// Check records msg for field when ok is false (first message per field wins).
func (v *Validator) Check(ok bool, field, msg string) {
	if ok {
		return
	}
	if _, exists := v.fields[field]; !exists {
		v.fields[field] = msg
	}
}

// Err returns nil when everything passed.
// We return a plain `error` and an explicit nil on purpose: returning a nil
// *apperr.Error as `error` would be NON-nil (Go's "typed nil" trap).
func (v *Validator) Err() error {
	if len(v.fields) == 0 {
		return nil
	}
	return apperr.Validation(v.fields)
}

// ---------- sanitizing (clean the input) ----------

// Text normalizes free text: control chars -> space, invisible/format chars
// (zero-width, right-to-left override) removed, whitespace collapsed, trimmed.
func Text(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsControl(r): // \n \t \x00 ...
			return ' '
		case unicode.Is(unicode.Cf, r): // U+200B, U+202E ...
			return -1 // drop it
		}
		return r
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

// Upper is for enum-like values: " active " -> "ACTIVE".
func Upper(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

// ---------- validating (allow-lists, never block-lists) ----------

func RuneLen(s string, min, max int) bool {
	n := utf8.RuneCountInString(s) // characters, not bytes
	return n >= min && n <= max
}

// Name allows letters (any language, so Tamil/Hindi work), combining marks,
// digits, space and  . - _ '   and nothing else.
func Name(s string) bool {
	for _, r := range s {
		ok := unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsDigit(r) ||
			strings.ContainsRune(" .-_'", r)
		if !ok {
			return false
		}
	}
	return true
}

func OneOf(s string, allowed ...string) bool { // "...string" = any number of strings
	for _, a := range allowed {
		if s == a {
			return true
		}
	}
	return false
}

var idempotencyRe = regexp.MustCompile(`^[A-Za-z0-9_-]{16,100}$`)

func IdempotencyKey(s string) bool { return idempotencyRe.MatchString(s) }

// ---------- money: strings in, integer paise out. NEVER float. ----------

// up to 15 digits, optional 1-2 decimals, no sign, no exponent, no leading zeros
var moneyRe = regexp.MustCompile(`^(0|[1-9][0-9]{0,14})(\.[0-9]{1,2})?$`)

// Money parses "5000.5" -> 500050. Caller still checks > 0 and <= limit.
func Money(s string) (int64, bool) {
	if !moneyRe.MatchString(s) {
		return 0, false
	}
	whole, frac, _ := strings.Cut(s, ".")
	for len(frac) < 2 {
		frac += "0"
	}
	w, _ := strconv.ParseInt(whole, 10, 64)
	f, _ := strconv.ParseInt(frac, 10, 64)
	return w*100 + f, true
}

func FormatMoney(paise int64) string {
	return fmt.Sprintf("%d.%02d", paise/100, paise%100)
}
