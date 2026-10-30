package project

import (
	"reflect"
	"testing"
	"time"
)

// HU-11, T005-T007: reglas puras de la modificación.

func day(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestDatesChanged(t *testing.T) {
	start, end := day("2026-03-01"), day("2026-06-30")
	buenosAires := time.FixedZone("ART", -3*60*60)
	cases := []struct {
		name             string
		newStart, newEnd time.Time
		want             bool
	}{
		{"mismas fechas", start, end, false},
		{"inicio distinto", day("2026-03-02"), end, true},
		{"fin distinto", start, day("2026-07-01"), true},
		{"misma fecha en otra zona horaria", time.Date(2026, 3, 1, 23, 0, 0, 0, buenosAires), end, false},
	}
	for _, c := range cases {
		if got := DatesChanged(start, end, c.newStart, c.newEnd); got != c.want {
			t.Errorf("%s: se esperaba %v, se obtuvo %v", c.name, c.want, got)
		}
	}
}

func TestMembershipChanges(t *testing.T) {
	cases := []struct {
		name                 string
		current, requested   []int
		wantAdded, wantRemov []int
	}{
		{"solo altas", []int{1, 2}, []int{3, 1, 2}, []int{3}, nil},
		{"solo bajas", []int{1, 2, 3}, []int{2}, nil, []int{1, 3}},
		{"altas y bajas", []int{5, 1}, []int{2, 5, 9}, []int{2, 9}, []int{1}},
		{"sin cambios", []int{2, 1}, []int{1, 2}, nil, nil},
		{"repetidos", []int{1, 1}, []int{3, 3, 1}, []int{3}, nil},
	}
	for _, c := range cases {
		added, removed := MembershipChanges(c.current, c.requested)
		if !reflect.DeepEqual(added, c.wantAdded) || !reflect.DeepEqual(removed, c.wantRemov) {
			t.Errorf("%s: se esperaba (%v, %v), se obtuvo (%v, %v)", c.name, c.wantAdded, c.wantRemov, added, removed)
		}
	}
}

func TestSprintsLeftOutside(t *testing.T) {
	oldStart, oldEnd := day("2026-03-01"), day("2026-06-30")
	sprint := func(n int, s, e string) SprintPeriod {
		return SprintPeriod{Number: n, StartDate: day(s), EndDate: day(e)}
	}
	s2 := sprint(2, "2026-06-01", "2026-06-14")
	cases := []struct {
		name             string
		newStart, newEnd string
		sprints          []SprintPeriod
		want             *SprintOutOfRangeError
	}{
		{"acortar el fin", "2026-03-01", "2026-06-10", []SprintPeriod{s2}, &SprintOutOfRangeError{2, DateFieldEnd}},
		{"adelantar el inicio", "2026-06-05", "2026-06-30", []SprintPeriod{s2}, &SprintOutOfRangeError{2, DateFieldStart}},
		{"ampliar", "2026-02-01", "2026-07-31", []SprintPeriod{s2}, nil},
		{"ya estaba fuera", "2026-03-01", "2026-07-31", []SprintPeriod{sprint(1, "2026-02-20", "2026-03-05")}, nil},
		{"borde exacto", "2026-06-01", "2026-06-14", []SprintPeriod{s2}, nil},
		{"fuera por los dos lados", "2026-06-05", "2026-06-10", []SprintPeriod{s2}, &SprintOutOfRangeError{2, DateFieldStart}},
		{"varios afectados: el de menor número", "2026-03-01", "2026-04-01",
			[]SprintPeriod{sprint(4, "2026-06-15", "2026-06-28"), s2, sprint(3, "2026-05-01", "2026-05-14")}, &SprintOutOfRangeError{2, DateFieldEnd}},
		{"un Sprint finalizado cuenta", "2026-03-01", "2026-03-10",
			[]SprintPeriod{sprint(1, "2026-03-02", "2026-03-15")}, &SprintOutOfRangeError{1, DateFieldEnd}},
		{"uno ya fuera y otro que queda fuera", "2026-03-01", "2026-06-10",
			[]SprintPeriod{sprint(1, "2026-02-20", "2026-03-05"), s2}, &SprintOutOfRangeError{2, DateFieldEnd}},
	}
	for _, c := range cases {
		got := SprintsLeftOutside(oldStart, oldEnd, day(c.newStart), day(c.newEnd), c.sprints)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: se esperaba %+v, se obtuvo %+v", c.name, c.want, got)
		}
	}
}
