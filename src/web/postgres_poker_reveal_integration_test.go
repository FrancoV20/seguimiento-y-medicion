package web

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// HU-05b, T016: rondas cerradas y votos revelados de PostgreSQLPokerCatalog, con los datos de «Alfa».

func pokerInt(v int) *int { return &v }

func revealed(id int64, name string, value *int) PokerRevealedVoteView {
	return PokerRevealedVoteView{MemberID: id, Name: name, Value: value}
}

// closedByID busca una ronda cerrada del tablero.
func closedByID(board PokerBoard, round int64) (PokerClosedRoundView, bool) {
	for _, c := range board.ClosedRounds {
		if c.ID == round {
			return c, true
		}
	}
	return PokerClosedRoundView{}, false
}

func alfaBoard(t *testing.T, f *alfaWebFixture, member int64) PokerBoard {
	t.Helper()
	board, err := NewPostgreSQLPokerCatalog(f.db).PokerBoard(context.Background(), f.alfa, member)
	if err != nil {
		t.Fatalf("PokerBoard devolvió error: %v", err)
	}
	return board
}

// T016 (E1, FR-007): de la más recientemente cerrada a la más antigua; empate: id mayor primero.
func TestPostgresPokerCatalog_RondasCerradas(t *testing.T) {
	f := newAlfaWebFixture(t, openContextTestDB(t))
	f.closeAt(t, f.rounds["Login"], alfaBase)
	f.closeAt(t, f.rounds["Pagos"], alfaBase.Add(5*time.Minute))
	same := alfaBase.Add(-time.Hour)
	f.closeAt(t, f.rounds["Reportes"], same)
	f.closeAt(t, f.rounds["Perfil"], same) // Perfil se creó después: id mayor

	board := alfaBoard(t, f, f.ana)
	var got []string
	for _, c := range board.ClosedRounds {
		got = append(got, c.StoryTitle)
	}
	if want := []string{"Pagos", "Login", "Perfil", "Reportes"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orden de «Rondas cerradas»: se esperaba %v, se obtuvo %v", want, got)
	}
	for i, title := range []string{"Pagos", "Login"} {
		c := board.ClosedRounds[i]
		if c.ID != f.rounds[title] || c.StoryID != f.stories[title] || c.Number != 1 {
			t.Errorf("%s: datos de la ronda cerrada inesperados: %+v", title, c)
		}
	}
}

// T016 (E1, E5, E7): valores por nombre, "No votó" como nil, y el mismo resultado para todos.
func TestPostgresPokerCatalog_VotosRevelados(t *testing.T) {
	f := newAlfaWebFixture(t, openContextTestDB(t))
	f.closeAt(t, f.rounds["Login"], alfaBase)
	f.closeAt(t, f.rounds["Perfil"], alfaBase.Add(time.Minute))

	board := alfaBoard(t, f, f.ana)
	login, ok := closedByID(board, f.rounds["Login"])
	want := []PokerRevealedVoteView{revealed(f.ana, "Ana", pokerInt(2)), revealed(f.beto, "Beto", pokerInt(13)), revealed(f.caro, "Caro", pokerInt(5))}
	if !ok || !reflect.DeepEqual(login.Votes, want) {
		t.Errorf("«Login»: se esperaban Ana 2, Beto 13 y Caro 5, se obtuvo %+v", login.Votes)
	}
	perfil, ok := closedByID(board, f.rounds["Perfil"])
	want = []PokerRevealedVoteView{revealed(f.ana, "Ana", pokerInt(5)), revealed(f.beto, "Beto", pokerInt(5)), revealed(f.caro, "Caro", nil)}
	if !ok || !reflect.DeepEqual(perfil.Votes, want) {
		t.Errorf("«Perfil»: se esperaban Ana 5, Beto 5 y Caro sin voto, se obtuvo %+v", perfil.Votes)
	}

	if beto := alfaBoard(t, f, f.beto); !reflect.DeepEqual(beto.ClosedRounds, board.ClosedRounds) {
		t.Errorf("Ana y Beto debían ver las mismas rondas cerradas:\nAna:  %+v\nBeto: %+v", board.ClosedRounds, beto.ClosedRounds)
	}
}

// T016 (R7): quien votó y ya no pertenece al proyecto sigue figurando con su valor; quien no votó
// y ya no pertenece, no figura.
func TestPostgresPokerCatalog_IntegranteQuitado(t *testing.T) {
	f := newAlfaWebFixture(t, openContextTestDB(t))
	f.removeMember(t, f.ana)
	f.removeMember(t, f.caro)
	f.closeAt(t, f.rounds["Login"], alfaBase)
	f.closeAt(t, f.rounds["Perfil"], alfaBase.Add(time.Minute))

	board := alfaBoard(t, f, f.beto)
	login, _ := closedByID(board, f.rounds["Login"])
	want := []PokerRevealedVoteView{revealed(f.ana, "Ana", pokerInt(2)), revealed(f.beto, "Beto", pokerInt(13)), revealed(f.caro, "Caro", pokerInt(5))}
	if !reflect.DeepEqual(login.Votes, want) {
		t.Errorf("«Login»: Ana y Caro votaron y debían seguir con 2 y 5, se obtuvo %+v", login.Votes)
	}
	perfil, _ := closedByID(board, f.rounds["Perfil"])
	want = []PokerRevealedVoteView{revealed(f.ana, "Ana", pokerInt(5)), revealed(f.beto, "Beto", pokerInt(5))}
	if !reflect.DeepEqual(perfil.Votes, want) {
		t.Errorf("«Perfil»: Caro no votó y ya no está en el proyecto, no debía figurar; se obtuvo %+v", perfil.Votes)
	}
}

// T016 (FR-011, SC-004): solo las rondas cerradas del proyecto activo; las rondas abiertas de HU-05a
// no cambian.
func TestPostgresPokerCatalog_SoloRondasCerradas(t *testing.T) {
	f := newAlfaWebFixture(t, openContextTestDB(t))
	before := alfaBoard(t, f, f.beto)
	if before.ClosedRounds == nil || len(before.ClosedRounds) != 0 {
		t.Fatalf("sin rondas cerradas en «Alfa», ClosedRounds debía ser vacío y no nil, se obtuvo %#v", before.ClosedRounds)
	}

	f.closeAt(t, f.rounds["Login"], alfaBase)
	after := alfaBoard(t, f, f.beto)
	if len(after.ClosedRounds) != 1 || after.ClosedRounds[0].ID != f.rounds["Login"] {
		t.Fatalf("solo «Login» debía estar entre las cerradas, se obtuvo %+v", after.ClosedRounds)
	}
	for _, c := range after.ClosedRounds {
		if c.ID == f.rounds["Pagos"] || c.ID == f.rounds["Agenda"] || c.ID == f.rounds["Tablero"] {
			t.Errorf("una ronda abierta o de otro proyecto apareció entre las cerradas: %+v", c)
		}
		if c.StoryID == f.stories["Pagos"] {
			t.Errorf("ningún voto revelado debía ser de «Pagos», que sigue abierta: %+v", c)
		}
	}

	var wantRounds []PokerRoundView
	for _, r := range before.Rounds {
		if r.ID != f.rounds["Login"] {
			wantRounds = append(wantRounds, r)
		}
	}
	if !reflect.DeepEqual(after.Rounds, wantRounds) || !reflect.DeepEqual(after.Stories, before.Stories) {
		t.Errorf("las rondas abiertas y sus participantes no debían cambiar salvo por «Login»:\nantes:   %+v\ndespués: %+v", before.Rounds, after.Rounds)
	}
}
