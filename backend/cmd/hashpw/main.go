package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"

	"golang.org/x/crypto/argon2"
)

func main() {
	password := flag.String("password", "", "password to hash")
	flag.Parse()
	if *password == "" {
		panic("password is required")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		panic(err)
	}
	hash := argon2.IDKey([]byte(*password), salt, 3, 64*1024, 4, 32)
	fmt.Printf("$argon2id$v=19$m=65536,t=3,p=4$%s$%s\n",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash))
}
