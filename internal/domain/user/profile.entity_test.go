package user_test

import (
	"strings"
	"testing"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/user"
)

func TestNewProfile_Valid(t *testing.T) {
	p, err := domain.NewProfile("Hanbin", "hanbin@example.com")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if p.Name() != "Hanbin" {
		t.Errorf("expected name 'Hanbin', got '%s'", p.Name())
	}
	if p.Email() != "hanbin@example.com" {
		t.Errorf("expected email 'hanbin@example.com', got '%s'", p.Email())
	}
	if p.CreatedAt().IsZero() {
		t.Error("expected non-zero createdAt")
	}
}

func TestNewProfile_EmptyName(t *testing.T) {
	_, err := domain.NewProfile("", "hanbin@example.com")
	if err != domain.ErrNameRequired {
		t.Errorf("expected ErrNameRequired, got: %v", err)
	}
}

func TestNewProfile_WhitespaceName(t *testing.T) {
	_, err := domain.NewProfile("   ", "hanbin@example.com")
	if err != domain.ErrNameRequired {
		t.Errorf("expected ErrNameRequired for whitespace-only name, got: %v", err)
	}
}

func TestNewProfile_EmptyEmail(t *testing.T) {
	_, err := domain.NewProfile("Hanbin", "")
	if err != domain.ErrEmailRequired {
		t.Errorf("expected ErrEmailRequired, got: %v", err)
	}
}

func TestNewProfile_InvalidEmail(t *testing.T) {
	cases := []string{"not-an-email", "@nodomain", "noat.com", "a@b"}
	for _, e := range cases {
		_, err := domain.NewProfile("Hanbin", e)
		if err != domain.ErrEmailInvalid {
			t.Errorf("email %q: expected ErrEmailInvalid, got: %v", e, err)
		}
	}
}

func TestNewProfile_NameTooLong(t *testing.T) {
	// 256 символов — превышает лимит
	long := strings.Repeat("a", 256)
	_, err := domain.NewProfile(long, "hanbin@example.com")
	if err != domain.ErrNameTooLong {
		t.Errorf("expected ErrNameTooLong, got: %v", err)
	}
}

func TestNewProfile_EmailNormalized(t *testing.T) {
	p, err := domain.NewProfile("Hanbin", "  HANBIN@Example.COM  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Email() != "hanbin@example.com" {
		t.Errorf("expected lowercase trimmed email, got '%s'", p.Email())
	}
}

func TestReconstitute(t *testing.T) {
	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	p := domain.Reconstitute(42, "Hanbin", "hanbin@example.com", created, updated)

	if p.ID() != 42 {
		t.Errorf("expected ID 42, got %d", p.ID())
	}
	if p.Name() != "Hanbin" {
		t.Errorf("expected name 'Hanbin', got '%s'", p.Name())
	}
	if !p.CreatedAt().Equal(created) {
		t.Errorf("createdAt mismatch")
	}
}

func TestSetName_UpdatesTimestamp(t *testing.T) {
	p, _ := domain.NewProfile("Old", "hanbin@example.com")
	before := p.UpdatedAt()

	time.Sleep(time.Millisecond)
	if err := p.SetName("New"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.UpdatedAt().After(before) {
		t.Error("expected updatedAt to advance after SetName")
	}
}
