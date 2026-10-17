package transfer

import (
	"crypto/rand"
	"fmt"
)

func NewReferenceNumber() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("TRF-%X", b), nil
}
