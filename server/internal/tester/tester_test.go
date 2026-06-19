package tester

import (
	"testing"
)

func TestParseUUID_Valid(t *testing.T) {
	expected := "550e8400-e29b-41d4-a716-446655440000"
	got := parseUUID(expected)

	if got == nil {
		t.Fatal("parseUUID returned nil")
	}
	if len(got) != 16 {
		t.Fatalf("expected 16 bytes, got %d", len(got))
	}

	// Verify the bytes match UUID format
	// UUID is: time_low (4) - time_mid (2) - time_hi_and_version (2) - clock_seq (2) - node (6)
	expectedBytes := []byte{
		0x55, 0x0e, 0x84, 0x00, // time_low
		0xe2, 0x9b, // time_mid
		0x41, 0xd4, // time_hi_and_version
		0xa7, 0x16, // clock_seq
		0x44, 0x66, 0x55, 0x44, 0x00, 0x00, // node
	}

	for i := range expectedBytes {
		if got[i] != expectedBytes[i] {
			t.Errorf("byte %d = 0x%02x, want 0x%02x", i, got[i], expectedBytes[i])
		}
	}
}

func TestParseUUID_AllZeros(t *testing.T) {
	uuid := "00000000-0000-0000-0000-000000000000"
	got := parseUUID(uuid)

	if got == nil {
		t.Fatal("parseUUID returned nil")
	}
	for i, b := range got {
		if b != 0 {
			t.Errorf("byte %d = 0x%02x, want 0x00", i, b)
		}
	}
}

func TestParseUUID_AllFs(t *testing.T) {
	uuid := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	got := parseUUID(uuid)

	if got == nil {
		t.Fatal("parseUUID returned nil")
	}
	for i, b := range got {
		if b != 0xff {
			t.Errorf("byte %d = 0x%02x, want 0xff", i, b)
		}
	}
}

func TestParseUUID_InvalidLength(t *testing.T) {
	tests := []string{
		"",                    // empty
		"too-short",           // way too short
		"550e8400-e29b-41d4",  // partial
		"550e8400e29b41d4a716446655440000", // no dashes, but 32 chars — won't match 36
	}

	for _, uuid := range tests {
		got := parseUUID(uuid)
		if got != nil {
			t.Errorf("parseUUID(%q) = %v, want nil", uuid, got)
		}
	}
}

func TestParseUUID_Random(t *testing.T) {
	// Test with a few randomly generated UUIDs
	uuids := []string{
		"a1b2c3d4-e5f6-7890-abcd-ef0123456789",
		"12345678-9abc-def0-1234-56789abcdef0",
	}

	for _, uuid := range uuids {
		got := parseUUID(uuid)
		if got == nil {
			t.Errorf("parseUUID(%q) = nil, want 16 bytes", uuid)
			continue
		}
		if len(got) != 16 {
			t.Errorf("parseUUID(%q) length = %d, want 16", uuid, len(got))
		}
	}
}

func TestUnhex(t *testing.T) {
	tests := []struct {
		input byte
		want  byte
	}{
		{'0', 0x0},
		{'9', 0x9},
		{'a', 0xa},
		{'f', 0xf},
		{'A', 0xA},
		{'F', 0xF},
		{'z', 0},  // invalid
		{' ', 0},  // invalid
		{0xFF, 0}, // invalid
	}

	for _, tt := range tests {
		got := unhex(tt.input)
		if got != tt.want {
			t.Errorf("unhex(%q) = 0x%x, want 0x%x", tt.input, got, tt.want)
		}
	}
}
