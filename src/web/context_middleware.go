package web

import (
	"net/http"
	"strings"
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