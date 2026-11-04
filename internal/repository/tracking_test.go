package repository_test

import (
	"testing"
	"smart-residential-backend/internal/repository"
)

func TestParseFechaFiltro(t *testing.T) {
	// Case 1: RFC3339 with UTC 'Z'
	strUTC := "2026-11-04T21:00:00.000Z"
	t1, err := repository.ParseFechaFiltro(strUTC)
	if err != nil {
		t.Fatalf("Error parsing UTC: %v", err)
	}
	if t1.Hour() != 21 {
		t.Errorf("Expected hour 21 in UTC, got %d", t1.Hour())
	}

	// Case 2: Local time without timezone (e.g. 15:00:00 in Nicaragua UTC-6)
	strLocal := "2026-11-04 15:00:00"
	t2, err := repository.ParseFechaFiltro(strLocal)
	if err != nil {
		t.Fatalf("Error parsing local: %v", err)
	}
	if t2.Hour() != 21 {
		t.Errorf("Expected 15:00 CST to convert to 21:00 UTC, got %d", t2.Hour())
	}

	// Case 3: Both t1 and t2 should represent the exact same UTC instant
	if !t1.Equal(t2) {
		t.Errorf("Expected t1 (%v) and t2 (%v) to be equal in UTC", t1, t2)
	}

	// Case 4: Format with 'T' and no timezone "2026-11-04T15:51:00"
	strLocalT := "2026-11-04T15:51:00"
	t3, err := repository.ParseFechaFiltro(strLocalT)
	if err != nil {
		t.Fatalf("Error parsing local with T: %v", err)
	}
	if t3.Hour() != 21 || t3.Minute() != 51 {
		t.Errorf("Expected 15:51 CST to convert to 21:51 UTC, got %v", t3)
	}
}
