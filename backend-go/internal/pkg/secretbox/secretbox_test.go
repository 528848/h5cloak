package secretbox

import "testing"

func TestRoundTrip(t *testing.T) {
	b, err := New("master")
	if err != nil {
		t.Fatal(err)
	}
	ct, err := b.Seal([]byte("sk_secret"))
	if err != nil {
		t.Fatal(err)
	}
	pt, err := b.Open(ct)
	if err != nil || string(pt) != "sk_secret" {
		t.Fatalf("open: %v %q", err, pt)
	}
	other, _ := New("other")
	if _, err := other.Open(ct); err == nil {
		t.Fatal("wrong key must fail")
	}
	if _, err := b.Open(ct[:5]); err == nil {
		t.Fatal("short ciphertext must fail")
	}
	if _, err := New(""); err == nil {
		t.Fatal("empty master")
	}
}
