# Investigación: HU-26 – Cookies firmadas

## Decisiones

1. **Firmar, no cifrar.**
   - Motivo: el valor es un identificador no secreto; el riesgo es la
     falsificación, no la lectura.
   - Alternativa descartada: sesiones guardadas en la base de datos (cambio de
     arquitectura grande y fuera del TP).

2. **HMAC-SHA256 con la biblioteca estándar.**
   - Motivo: sin dependencias nuevas y suficiente para integridad.
   - Alternativa descartada: biblioteca externa de sesiones.

3. **Ligar el nombre de la cookie a la firma.**
   - Motivo: sin eso, el valor firmado de `ctx_proyecto` serviría como
     `ctx_integrante`.

4. **Clave global del proceso.**
   - Motivo: `readIDCookie` y `setContextCookie` no reciben el servidor; pasar la
     clave por parámetro cambiaría esas funciones, `clearContextCookie`,
     `RegisterContext` y las pruebas que las llaman, con riesgo alto de
     conflictos con ramas abiertas.
   - Costo: estado global; se acota a "se fija una vez al arrancar" y se
     restaura en las pruebas.

5. **Clave temporal si falta `COOKIE_SECRET`.**
   - Motivo: la demo del 2/11 y el desarrollo local no deben depender de
     configuración.
   - Costo: las sesiones se pierden al reiniciar; se avisa en el registro.

6. **Clave corta: la aplicación no arranca.**
   - Motivo: una clave débil da falsa seguridad y es un error de configuración
     evidente. Se comprueba antes de abrir el puerto o la base de datos.

7. **Cookie inválida = cookie manipulada de HU-20b.**
   - Motivo: el middleware ya la borra en silencio y sigue sin contexto; los
     escenarios 2, 3 y 5 reutilizan ese camino ya probado.

8. **`flash` fuera de alcance.**
   - Motivo: no concede acceso; su texto lo define el servidor y se escapa
     (ya cubierto por pruebas de HU-20b).

9. **Límite del alcance.** Firmar impide inventar un contexto que el servidor no
   emitió, pero no impide elegir cualquier proyecto desde la pantalla. Se dice
   así en la presentación.

## Datos verificados en `main` (27/10/2026)

- Nombres de cookies y funciones: ver `plan.md`.
- Pruebas que dependen del valor crudo de la cookie:
  `src/web/context_test.go` (`TestReadIDCookie_Valores`,
  `TestSetContextCookie_AtributosDeSesion`), `context_handlers_test.go`
  (`idCookie` y la comparación con `id64(...)` tras una caída transitoria),
  `context_member_test.go` (comparación de `ck.Value` con `id64(fx.ana)`),
  `projects_handlers_test.go` (cookies armadas con `fmt.Sprint`) y
  `test/context_bdd_test.go` (`cookieID`).

## Datos a reconfirmar en T001 (el código pudo cambiar)

- Que no hayan entrado a `main` pruebas nuevas que fabriquen cookies de contexto
  a mano (`git grep -n "CookieProject\|CookieMember" -- '*_test.go' test/`).
- La firma actual de la preparación de pruebas de `test/context_bdd_test.go`
  (`nuevaSeleccionApp`, `formNavegador`, `poner`, `valorCookie`).

## Preguntas abiertas

Ninguna.