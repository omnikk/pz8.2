package service

import "testing"

func TestLogin_ValidCredentials(t *testing.T) {
	s := New()
	token, ok := s.Login("student", "student")
	if !ok {
		t.Fatal("expected login to succeed with valid credentials")
	}
	if token != "demo-token" {
		t.Errorf("expected token %q, got %q", "demo-token", token)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	s := New()
	_, ok := s.Login("student", "wrong-password")
	if ok {
		t.Fatal("expected login to fail with wrong password")
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	s := New()
	_, ok := s.Login("hacker", "student")
	if ok {
		t.Fatal("expected login to fail with unknown user")
	}
}

func TestVerify_ValidToken(t *testing.T) {
	s := New()
	subject, valid := s.Verify("demo-token")
	if !valid {
		t.Fatal("expected token to be valid")
	}
	if subject != "student" {
		t.Errorf("expected subject %q, got %q", "student", subject)
	}
}

func TestVerify_InvalidToken(t *testing.T) {
	s := New()
	_, valid := s.Verify("not-a-real-token")
	if valid {
		t.Fatal("expected invalid token to be rejected")
	}
}
