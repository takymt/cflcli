package cmd

import "testing"

func TestPageListLimitBounds(t *testing.T) {
	if err := validatePageListLimit(0); err == nil {
		t.Fatalf("expected error for limit 0")
	}
	if err := validatePageListLimit(251); err == nil {
		t.Fatalf("expected error for limit 251")
	}
	if err := validatePageListLimit(1); err != nil {
		t.Fatalf("unexpected error for limit 1: %v", err)
	}
	if err := validatePageListLimit(250); err != nil {
		t.Fatalf("unexpected error for limit 250: %v", err)
	}
}
