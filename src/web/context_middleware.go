package web

import (
	"net/http"
	"strings"
	"net/url"
)

// contextMiddleware resuelve el contexto de trabajo en cada solicitud (HU-20b): lee las cookies,
// las valida contra la base con una sola consulta y completa el ActiveContext.
//   - Nunca hay integrante sin proyecto: sin proyecto válido se descartan ambas cookies.
//   - Una cookie manipulada (no numérica, fuera de rango) se descarta en silencio.
//   - Un proyecto que ya no existe, o un integrante que no pertenece al proyecto, se descartan
//     con una advertencia que se muestra en esta misma respuesta.
//   - Si la base no responde no se borra nada y se responde 503.
func contextMiddleware(store ContextStore, rd *Renderer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/static/") {
				next.ServeHTTP(w, r) // los archivos estáticos no dependen del contexto
				return
			}
			projectID, projectPresent := readIDCookie(r, CookieProject)
			memberID, memberPresent := readIDCookie(r, CookieMember)

			if !projectPresent || projectID == 0 {
				if projectPresent {
					clearContextCookie(w, r, CookieProject)
				}
				if memberPresent {
					clearContextCookie(w, r, CookieMember)
				}
				next.ServeHTTP(w, r)
				return
			}

			resolved, err := store.ResolveContext(r.Context(), projectID, memberID)
			if err != nil {
				rd.Fail(w, r, err)
				return
			}
			ctx := r.Context()
			if !resolved.ProjectExists {
				clearContextCookie(w, r, CookieProject)
				if memberPresent {
					clearContextCookie(w, r, CookieMember)
				}
				ctx = withPendingFlash(ctx, &Flash{Kind: FlashWarning, Text: MsgProjectGone})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			ac := &ActiveContext{ProjectID: projectID, ProjectName: resolved.ProjectName}
			switch {
			case !memberPresent:
				// sin integrante identificado
			case memberID == 0:
				clearContextCookie(w, r, CookieMember) // valor manipulado
			case !resolved.MemberBelongs:
				clearContextCookie(w, r, CookieMember)
				ctx = withPendingFlash(ctx, &Flash{Kind: FlashWarning, Text: MsgMemberGone})
			default:
				ac.MemberID, ac.MemberName = memberID, resolved.MemberName
			}
			next.ServeHTTP(w, r.WithContext(WithActiveContext(ctx, ac)))
		})
	}
}
// RequireContext protege una sección según RequiresProject y RequiresMember (HU-20b, T052).
// Si falta el proyecto o el integrante, redirige con 303 a elegirlo, con un mensaje, y lleva el
// destino de regreso para volver a esta sección. Una sección sin requisitos pasa sin cambios.
// Un integrante siempre exige proyecto: sin proyecto se pide primero el proyecto.
func RequireContext(section Section) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if !section.RequiresProject && !section.RequiresMember {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ac := ActiveContextFrom(r.Context())
			hasProject := ac != nil && ac.ProjectID != 0
			hasMember := hasProject && ac.MemberID != 0
			switch {
			case !hasProject:
				SetFlash(w, r, FlashWarning, MsgProjectRequired)
				RedirectSeeOther(w, r, pathSelectProject+"?volver_a="+url.QueryEscape(returnDestination(r, section)))
			case section.RequiresMember && !hasMember:
				SetFlash(w, r, FlashWarning, MsgMemberRequired)
				RedirectSeeOther(w, r, pathSelectMember+"?volver_a="+url.QueryEscape(returnDestination(r, section)))
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}

// returnDestination es la dirección a la que se vuelve después de elegir: la misma pantalla si
// se pidió con GET o HEAD; si fue un envío (POST...), la sección de origen, para volver por GET.
// Los datos del formulario no se conservan (limitación aceptada, ver research.md).
func returnDestination(r *http.Request, section Section) string {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return r.URL.RequestURI()
	}
	return strings.TrimSpace(section.Path)
}