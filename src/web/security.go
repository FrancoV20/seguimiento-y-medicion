package web

import "net/http"

// CrossOriginGuard devuelve un middleware que rechaza con 403 los envíos no seguros
// (POST, PUT, DELETE...) originados en otro sitio web, usando http.CrossOriginProtection
// (cabeceras Sec-Fetch-Site y Origin). GET, HEAD y OPTIONS siempre pasan, y las solicitudes
// sin esas cabeceras (curl, pruebas) se permiten. La respuesta es la página de error común
// con el código 403 y sin ejecutar ninguna acción (HU-20b, T024).
func CrossOriginGuard(rd *Renderer) func(http.Handler) http.Handler {
	cop := http.NewCrossOriginProtection()
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rd.RenderError(w, r, http.StatusForbidden, TitleForbidden, MsgForbidden)
	}))
	return cop.Handler
}
