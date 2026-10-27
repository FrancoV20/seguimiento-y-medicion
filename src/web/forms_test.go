package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestFormulario_ErrorJuntoAlCampoYConservaDatos(t *testing.T) {
	app := newFormsApp(t)
	client := app.client(true)

	resp, body := app.post(t, client, "/prueba/form",
		url.Values{"nombre": {""}, "detalle": {"Texto que no se pierde"}})

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("se esperaba 422, se obtuvo %d", resp.StatusCode)
	}
	iNombre := strings.Index(body, `name="nombre"`)
	iError := strings.Index(body, "El nombre es obligatorio")
	iDetalle := strings.Index(body, `name="detalle"`)
	if iNombre < 0 || iError < iNombre || iDetalle < iError {
		t.Errorf("el error debe estar entre el campo nombre y el campo detalle (posiciones %d, %d, %d)", iNombre, iError, iDetalle)
	}
	if !strings.Contains(body, `value="Texto que no se pierde"`) {
		t.Errorf("el dato ingresado no se conservó")
	}
	if app.saved.Load() != 0 {
		t.Errorf("no debía guardarse nada, se guardaron %d", app.saved.Load())
	}
}

func TestFormulario_FalloDeGuardadoConservaDatosYNoMuestraExito(t *testing.T) {
	app := newFormsApp(t)
	client := app.client(true)

	resp, body := app.post(t, client, "/prueba/form",
		url.Values{"nombre": {"Proyecto X"}, "detalle": {"Detalle Y"}, "falla": {"1"}})

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("se esperaba 500, se obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(body, MsgOperationFailed) {
		t.Errorf("falta el mensaje %q", MsgOperationFailed)
	}
	if !strings.Contains(body, `value="Proyecto X"`) || !strings.Contains(body, `value="Detalle Y"`) {
		t.Errorf("los datos ingresados no se conservaron")
	}
	if strings.Contains(body, formsSuccessText) {
		t.Errorf("no debía mostrarse el mensaje de éxito")
	}
	if app.saved.Load() != 0 {
		t.Errorf("no debía guardarse nada")
	}
}

func TestFormulario_EnvioValidoRedirigeYNoSeDuplicaAlRecargar(t *testing.T) {
	app := newFormsApp(t)
	client := app.client(false)

	resp, _ := app.post(t, client, "/prueba/form", url.Values{"nombre": {"Proyecto X"}})

	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/prueba/resultado" {
		t.Fatalf("se esperaba 303 a /prueba/resultado, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if app.saved.Load() != 1 {
		t.Fatalf("se esperaba 1 registro, hay %d", app.saved.Load())
	}
	_, first := app.get(t, client, "/prueba/resultado")
	if !strings.Contains(first, formsSuccessText) {
		t.Errorf("la página de resultado debía mostrar el mensaje de éxito")
	}
	_, reload := app.get(t, client, "/prueba/resultado")
	if strings.Contains(reload, formsSuccessText) {
		t.Errorf("al recargar, el mensaje no debía repetirse")
	}
	if app.saved.Load() != 1 {
		t.Errorf("al recargar no debía duplicarse el registro: hay %d", app.saved.Load())
	}
}

func TestFlash_SoloSeMuestraEnLaPrimeraPantalla(t *testing.T) {
	app := newFormsApp(t)
	client := app.client(false)
	app.post(t, client, "/prueba/form", url.Values{"nombre": {"Proyecto X"}})

	_, first := app.get(t, client, "/libre")
	_, second := app.get(t, client, "/prueba/resultado")

	if !strings.Contains(first, formsSuccessText) {
		t.Errorf("la primera pantalla debía mostrar el mensaje")
	}
	if strings.Contains(second, formsSuccessText) {
		t.Errorf("la segunda pantalla no debía repetirlo")
	}
}
