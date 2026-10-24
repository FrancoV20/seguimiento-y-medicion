package sprint

import "testing"

// HU-14 (T005): regla de transiciones de estado de una historia del Sprint activo.

// FR-004, FR-005, FR-007: las tres transiciones permitidas.
func TestCanTransition_LasTresPermitidas(t *testing.T) {
	permitidas := []struct{ origen, destino StoryStatus }{
		{EnSprint, InProgress},
		{InProgress, Done},
		{Done, InProgress},
	}
	for _, p := range permitidas {
		t.Run(string(p.origen)+" a "+string(p.destino), func(t *testing.T) {
			if !CanTransition(p.origen, p.destino) {
				t.Errorf("CanTransition(%q, %q) = false, se esperaba true", p.origen, p.destino)
			}
		})
	}
}

// FR-010: cualquier otro par se rechaza (En Sprint a Hecho, En progreso a En Sprint,
// Hecho a En Sprint, cualquier destino Pendiente y las repetidas).
func TestCanTransition_RechazaCualquierOtroPar(t *testing.T) {
	estados := []StoryStatus{Pendiente, EnSprint, InProgress, Done}
	permitidas := map[[2]StoryStatus]bool{
		{EnSprint, InProgress}: true,
		{InProgress, Done}:     true,
		{Done, InProgress}:     true,
	}
	for _, origen := range estados {
		for _, destino := range estados {
			if permitidas[[2]StoryStatus{origen, destino}] {
				continue
			}
			t.Run(string(origen)+" a "+string(destino), func(t *testing.T) {
				if CanTransition(origen, destino) {
					t.Errorf("CanTransition(%q, %q) = true, se esperaba false", origen, destino)
				}
			})
		}
	}
}

// FR-003: cada estado ofrece una sola acción; Pendiente no ofrece ninguna.
func TestNextAction_UnaAccionPorEstado(t *testing.T) {
	casos := []struct {
		estado StoryStatus
		accion Action
		hay    bool
	}{
		{EnSprint, ActionStart, true},
		{InProgress, ActionComplete, true},
		{Done, ActionReopen, true},
		{Pendiente, "", false},
	}
	for _, c := range casos {
		t.Run(string(c.estado), func(t *testing.T) {
			accion, ok := NextAction(c.estado)
			if ok != c.hay || accion != c.accion {
				t.Errorf("NextAction(%q) = (%q, %v), se esperaba (%q, %v)", c.estado, accion, ok, c.accion, c.hay)
			}
		})
	}
}

// FR-004, FR-005, FR-007: origen y destino de cada acción.
func TestActionTransition_OrigenYDestinoDeCadaAccion(t *testing.T) {
	casos := []struct {
		accion          Action
		origen, destino StoryStatus
	}{
		{ActionStart, EnSprint, InProgress},
		{ActionComplete, InProgress, Done},
		{ActionReopen, Done, InProgress},
	}
	for _, c := range casos {
		t.Run(string(c.accion), func(t *testing.T) {
			origen, destino, ok := ActionTransition(c.accion)
			if !ok || origen != c.origen || destino != c.destino {
				t.Errorf("ActionTransition(%q) = (%q, %q, %v), se esperaba (%q, %q, true)",
					c.accion, origen, destino, ok, c.origen, c.destino)
			}
		})
	}
}

// FR-015: una acción desconocida no tiene origen ni destino.
func TestActionTransition_AccionDesconocida(t *testing.T) {
	for _, accion := range []Action{"", "borrar"} {
		t.Run("accion "+string(accion), func(t *testing.T) {
			origen, destino, ok := ActionTransition(accion)
			if ok || origen != "" || destino != "" {
				t.Errorf("ActionTransition(%q) = (%q, %q, %v), se esperaba (\"\", \"\", false)", accion, origen, destino, ok)
			}
		})
	}
}

// FR-003 y punto 5 (decisión del 24/10/2026): las acciones salen de la misma regla.
// El origen de cada acción puede pasar al destino y su única acción es esa.
func TestAcciones_SonCoherentesConLaRegla(t *testing.T) {
	for _, accion := range []Action{ActionStart, ActionComplete, ActionReopen} {
		t.Run(string(accion), func(t *testing.T) {
			origen, destino, ok := ActionTransition(accion)
			if !ok {
				t.Fatalf("ActionTransition(%q) no reconoce la acción", accion)
			}
			if !CanTransition(origen, destino) {
				t.Errorf("la acción %q va de %q a %q, que CanTransition no permite", accion, origen, destino)
			}
			if siguiente, hay := NextAction(origen); !hay || siguiente != accion {
				t.Errorf("NextAction(%q) = (%q, %v), se esperaba (%q, true)", origen, siguiente, hay, accion)
			}
		})
	}
}
