package main

import "golang.org/x/crypto/bcrypt"
import "testing"

func TestUserPassword(t *testing.T) {
	u := User{}

	if u.ComparePassword("") {
		t.Error("no password set, but succeeded comparison")
	}

	if !u.SetPassword("", "hunter2") {
		t.Fatal("tried to set initial password, but failed.")
	}

	if !u.ComparePassword("hunter2") {
		t.Error("comparison with set password failed.")
	}

	if u.ComparePassword("2hunter") {
		t.Error("comparison with non-password succeeded.")
	}
}

func BenchmarkBcryptCost(b *testing.B) {
	var new = []byte("hunter2")
	for i := 0; i < b.N; i++ {
		_, _ = bcrypt.GenerateFromPassword(new, bcryptCost)
	}
}
