package auth

import "testing"

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := hashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !verifyPassword("correct horse battery staple", hash) {
		t.Fatal("correct password did not verify")
	}
	if verifyPassword("wrong password", hash) {
		t.Fatal("wrong password verified")
	}
	if verifyPassword("password", hash[:len(hash)-1]+"x") {
		t.Fatal("tampered hash verified")
	}
}

func TestPasswordHashUsesRandomSalt(t *testing.T) {
	first, err := hashPassword("same password")
	if err != nil {
		t.Fatal(err)
	}
	second, err := hashPassword("same password")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("same password produced identical hashes")
	}
}
