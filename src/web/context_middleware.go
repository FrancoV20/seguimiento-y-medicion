package web

import (
	"net/http"
	"strings"
)

// contextMiddleware resuelve el contexto de trabajo en cada solicitud (HU-20b, T036): lee las
// cookies, las valida contra la base con una sola consulta y completa el ActiveContext.
// Una cookie manipulada se trata como ausencia y se borra; un proyecto que ya no existe se
// borra con una advertencia; si la base no responde, no se borra nada y se responde 503.
func contextMiddleware(store ContextStore, rd *Renderer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/static/") {
				next.ServeHTTP(w, r) // los archivos estáticos no dependen del contexto
				return
			}
			projectID, present := readIDCookie(r, CookieProject)
			if !present {
				next.ServeHTTP(w, r)
				return
			}
			if projectID == 0 {
				clearContextCookie(w, r, CookieProject)
				next.ServeHTTP(w, r)
				return
			}
			memberID, _ := readIDCookie(r, CookieMember)
			resolved, err := store.ResolveContext(r.Context(), projectID, memberID)
			if err != nil {
				rd.Fail(w, r, err)
				return
			}
			if !resolved.ProjectExists {
				clearContextCookie(w, r, CookieProject)
				ctx := withPendingFlash(r.Context(), &Flash{Kind: FlashWarning, Text: MsgProjectGone})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			ac := &ActiveContext{ProjectID: projectID, ProjectName: resolved.ProjectName}
			if resolved.MemberBelongs {
				ac.MemberID, ac.MemberName = memberID, resolved.MemberName
			}
			next.ServeHTTP(w, r.WithContext(WithActiveContext(r.Context(), ac)))
		})
	}
}