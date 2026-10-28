package web

import (
	"reflect"
	"strings"
	"testing"
)

// HU-05a, T021: el modelo de vista de un participante no tiene dónde guardar un valor ajeno.
func TestPokerViewModel_SinCampoDeValorAjeno(t *testing.T) {
	typ := reflect.TypeOf(PokerParticipantView{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name := strings.ToLower(field.Name)
		if strings.Contains(name, "value") || strings.Contains(name, "vote") && field.Name != "HasVoted" {
			t.Errorf("PokerParticipantView no debe tener el campo %s", field.Name)
		}
		switch field.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64, reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
			if field.Name != "MemberID" {
				t.Errorf("PokerParticipantView no debe tener el campo numérico o compuesto %s (%s)", field.Name, field.Type)
			}
		}
	}
}
