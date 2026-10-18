package sprint

import "testing"

func TestFormatSprintID(t *testing.T) {
	tests := []struct {
		number int64
		want   string
	}{
		{number: 1, want: "Sprint 1"},
		{number: 2, want: "Sprint 2"},
		{number: 27, want: "Sprint 27"},
	}
	for _, tt := range tests {
		got, err := formatSprintID(tt.number)
		if err != nil {
			t.Fatalf("formatSprintID(%d) devolvió error: %v", tt.number, err)
		}
		if got != tt.want {
			t.Errorf("formatSprintID(%d) = %q, se esperaba %q", tt.number, got, tt.want)
		}
	}
}

func TestFormatSprintIDRejectsNonPositiveNumber(t *testing.T) {
	for _, number := range []int64{0, -1} {
		if _, err := formatSprintID(number); err == nil {
			t.Errorf("formatSprintID(%d) debía rechazar un número no positivo", number)
		}
	}
}
