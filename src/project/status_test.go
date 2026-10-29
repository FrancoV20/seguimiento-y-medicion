package project

import (
	"testing"
	"time"
)

var statusLocalZone = time.FixedZone("UTC-3", -3*60*60)

func statusUTC(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestDateStatusTexts(t *testing.T) {
	if string(StatusNotStarted) != "No iniciado" ||
		string(StatusInProgress) != "En curso" ||
		string(StatusFinished) != "Finalizado" {
		t.Fatalf("textos inesperados: %q, %q, %q", StatusNotStarted, StatusInProgress, StatusFinished)
	}
}

func TestStatusOn(t *testing.T) {
	loc := time.Local
	tests := []struct {
		name       string
		start, end time.Time
		today      time.Time
		want       DateStatus
	}{
		{"Alfa en curso", statusUTC(2026, 10, 1), statusUTC(2026, 11, 30), time.Date(2026, 10, 28, 10, 0, 0, 0, loc), StatusInProgress},
		{"Beta no iniciado", statusUTC(2026, 12, 1), statusUTC(2027, 1, 31), time.Date(2026, 10, 28, 10, 0, 0, 0, loc), StatusNotStarted},
		{"Gamma finalizado", statusUTC(2026, 8, 1), statusUTC(2026, 9, 30), time.Date(2026, 10, 28, 10, 0, 0, 0, loc), StatusFinished},
		{"Delta empieza hoy 00:00", statusUTC(2026, 10, 28), statusUTC(2026, 12, 15), time.Date(2026, 10, 28, 0, 0, 0, 0, loc), StatusInProgress},
		{"Delta empieza hoy 23:59", statusUTC(2026, 10, 28), statusUTC(2026, 12, 15), time.Date(2026, 10, 28, 23, 59, 0, 0, loc), StatusInProgress},
		{"Épsilon termina hoy 00:00", statusUTC(2026, 9, 1), statusUTC(2026, 10, 28), time.Date(2026, 10, 28, 0, 0, 0, 0, loc), StatusInProgress},
		{"Épsilon termina hoy 23:59", statusUTC(2026, 9, 1), statusUTC(2026, 10, 28), time.Date(2026, 10, 28, 23, 59, 0, 0, loc), StatusInProgress},
		{"día siguiente al fin", statusUTC(2026, 9, 1), statusUTC(2026, 10, 28), time.Date(2026, 10, 29, 0, 0, 0, 0, loc), StatusFinished},
		{"día anterior al inicio", statusUTC(2026, 10, 28), statusUTC(2026, 12, 15), time.Date(2026, 10, 27, 23, 59, 0, 0, loc), StatusNotStarted},
		// 27/10 22:00 en UTC-3 ya es 28/10 01:00 UTC: cuenta el día local.
		{"UTC-3 un día antes", statusUTC(2026, 10, 28), statusUTC(2026, 12, 15), time.Date(2026, 10, 27, 22, 0, 0, 0, statusLocalZone), StatusNotStarted},
		{"UTC-3 el día de inicio", statusUTC(2026, 10, 28), statusUTC(2026, 12, 15), time.Date(2026, 10, 28, 0, 0, 0, 0, statusLocalZone), StatusInProgress},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := StatusOn(tc.start, tc.end, tc.today); got != tc.want {
				t.Fatalf("StatusOn() = %q, esperado %q", got, tc.want)
			}
		})
	}
}

func TestProgressPercentage(t *testing.T) {
	tests := []struct {
		done, total int
		want        string
	}{
		{4, 8, "50.00"},
		{3, 3, "100.00"},
		{0, 0, "0.00"},
		{0, 5, "0.00"},
		{3, -1, "0.00"},
		{2, 3, "66.67"},
		{1, 3, "33.33"},
		{1, 8, "12.50"},
		{4, 7, "57.14"},
		{1, 6, "16.67"},
		{7, 11, "63.64"},
		{1, 32, "3.13"}, // 3,125: mitad exacta, sube
	}
	for _, tc := range tests {
		if got := ProgressPercentage(tc.done, tc.total); got != tc.want {
			t.Errorf("ProgressPercentage(%d, %d) = %q, esperado %q", tc.done, tc.total, got, tc.want)
		}
	}
}