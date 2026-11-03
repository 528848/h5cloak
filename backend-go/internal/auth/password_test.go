package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestArgonRoundTrip(t *testing.T) {
	h, err := HashPassword("Secret123")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(h, "Secret123") {
		t.Fatal("expected match")
	}
	if VerifyPassword(h, "Secret124") {
		t.Fatal("expected mismatch")
	}
	if NeedsRehash(h) {
		t.Fatal("argon hash should not need rehash")
	}
}

func TestBcryptCompat(t *testing.T) {
	// V1 Laravel 使用 $2y$；Go bcrypt 接受该前缀
	h, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	h[2] = 'y'
	if !VerifyPassword(string(h), "admin123") {
		t.Fatal("bcrypt $2y$ must verify")
	}
	if !NeedsRehash(string(h)) {
		t.Fatal("bcrypt should need rehash")
	}
}

func TestPasswordPolicy(t *testing.T) {
	if err := ValidatePasswordPolicy("short1"); err == nil {
		t.Fatal("too short should fail")
	}
	if err := ValidatePasswordPolicy("onlyletters"); err == nil {
		t.Fatal("letters only should fail")
	}
	if err := ValidatePasswordPolicy("12345678"); err == nil {
		t.Fatal("digits only should fail")
	}
	if err := ValidatePasswordPolicy("Good1234"); err != nil {
		t.Fatal(err)
	}
}
