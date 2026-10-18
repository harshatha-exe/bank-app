package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const dummyHash = "$argon2id$v=19$m=65536,t=3,p=4$XBZuxhlBdjEIA8YErz3Yhg$qACEeur7NKLfS3a31NapdPuNYFPjY8JP12YF1TJlZFs"

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=4$%s$%s",
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}

func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return false
	}
	values := make(map[string]uint64, 3)
	for _, param := range params {
		pair := strings.SplitN(param, "=", 2)
		if len(pair) != 2 {
			return false
		}
		value, err := strconv.ParseUint(pair[1], 10, 32)
		if err != nil {
			return false
		}
		values[pair[0]] = value
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, uint32(values["t"]), uint32(values["m"]), uint8(values["p"]), uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

// HashPassword is the exported entry point for other packages (admin).
func HashPassword(password string) (string, error) { return hashPassword(password) }
