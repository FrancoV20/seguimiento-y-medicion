package project

import (
	"sort"
	"time"
)

// calendarDay devuelve la fecha de calendario sin hora ni zona horaria.
func calendarDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// DatesChanged indica si cambió alguna de las fechas del proyecto, comparando solo año, mes y
// día (R7): la revisión de Sprints se hace solo cuando cambian las fechas.
func DatesChanged(oldStart, oldEnd, newStart, newEnd time.Time) bool {
	return !calendarDay(oldStart).Equal(calendarDay(newStart)) || !calendarDay(oldEnd).Equal(calendarDay(newEnd))
}

// MembershipChanges devuelve los integrantes que se agregan y los que se quitan, ordenados y
// sin repetidos.
func MembershipChanges(current, requested []int) (added, removed []int) {
	cur := toSet(current)
	req := toSet(requested)
	for id := range req {
		if !cur[id] {
			added = append(added, id)
		}
	}
	for id := range cur {
		if !req[id] {
			removed = append(removed, id)
		}
	}
	sort.Ints(added)
	sort.Ints(removed)
	return added, removed
}

func toSet(ids []int) map[int]bool {
	set := make(map[int]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// SprintsLeftOutside devuelve el Sprint de menor número que estaba dentro de las fechas
// actuales y queda fuera de las nuevas (R6), o nil. Un Sprint que ya estaba fuera no bloquea.
// El campo es el inicio si el Sprint empieza antes del nuevo inicio y, si no, el fin.
func SprintsLeftOutside(oldStart, oldEnd, newStart, newEnd time.Time, sprints []SprintPeriod) *SprintOutOfRangeError {
	oldStart, oldEnd = calendarDay(oldStart), calendarDay(oldEnd)
	newStart, newEnd = calendarDay(newStart), calendarDay(newEnd)

	var found *SprintOutOfRangeError
	for _, s := range sprints {
		start, end := calendarDay(s.StartDate), calendarDay(s.EndDate)
		wasInside := !start.Before(oldStart) && !end.After(oldEnd)
		leftOutside := start.Before(newStart) || end.After(newEnd)
		if !wasInside || !leftOutside {
			continue
		}
		if found != nil && found.Number <= s.Number {
			continue
		}
		field := DateFieldEnd
		if start.Before(newStart) {
			field = DateFieldStart
		}
		found = &SprintOutOfRangeError{Number: s.Number, Field: field}
	}
	return found
}
