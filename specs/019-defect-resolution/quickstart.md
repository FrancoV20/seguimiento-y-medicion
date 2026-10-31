# Guía rápida: HU-19 – Resolver y reabrir defectos

## Requisitos previos

- HU-06 mergeada en `main`.
- PostgreSQL de pruebas disponible (el mismo que usan las demás historias).
- Rama: `git checkout HU-19` (creada desde `main` actualizado)

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run HU19 ./test/... ./src/web/...
go test -count=1 -p 1 -cover ./src/defect/... ./src/web/...
```

## Verificación manual (T058)

Con la base al día (`migrate ... up`), `DATABASE_URL` y `COOKIE_SECRET` definidas, ejecutar
`go run ./src/cmd/server`, abrir `http://localhost:8080`, elegir un proyecto desde **Cambiar
proyecto** y abrir **Defectos**. Los escenarios 4 y 6 necesitan datos o fallas que la pantalla no
permite generar.

| # | Escenario | Pasos | Resultado esperado | ✓ |
|---|---|---|---|---|
| 1 | Resolución exitosa | Con un Sprint activo, registrar un defecto y pulsar **Resolver** | "Resuelto", "Sprint N" en Sprint de resolución y `Defecto resuelto en el Sprint N` una sola vez (recargar no lo repite) | ✓ |
| 2 | Sin Sprint activo | Cerrar el Sprint (HU-15) y pulsar **Resolver** en un defecto "Abierto" | `No hay un Sprint activo para registrar la resolución del defecto`; sigue "Abierto" | ✓ |
| 3 | Reabrir | Pulsar **Reabrir** en un defecto "Resuelto" (con y sin Sprint activo) | `Defecto reabierto`; vuelve a "Abierto" con `—` | ✓ (sin Sprint activo, después de T063) |
| 4 | Sprint anterior | En la base: defecto detectado en el Sprint 3 (finalizado) con el Sprint 2 activo; pulsar **Resolver** | `El Sprint de resolución no puede ser anterior al Sprint en que se detectó el defecto`; sigue "Abierto" | Cubierto por pruebas automáticas: `TestHU19_SprintAnteriorAlDeDeteccion` y `TestPostgresResolve_Anterior` (la pantalla no permite generar el dato) |
| 5 | Inexistente | Cambiar el valor oculto `defecto` con las herramientas del navegador (F12) a un número que no exista y pulsar **Resolver** | `El defecto no existe`; nada cambia | ✓ |
| 6 | Error al guardar | Con la pantalla abierta, detener la base (`docker compose stop`) y pulsar **Resolver** | Página de error sin detalles técnicos; al levantar la base, el defecto no cambió | ✓ (se muestra "Servicio no disponible" y el defecto no cambia) |
| 7 | Ya resuelto | Abrir la pantalla en dos pestañas y pulsar **Resolver** del mismo defecto en ambas | La segunda muestra `El defecto ya está resuelto`; conserva su Sprint de resolución | ✓ |
| 8 | Ya abierto | Abrir la pantalla en dos pestañas y pulsar **Reabrir** del mismo defecto en ambas | La segunda muestra `El defecto ya está abierto` | ✓ |
| 9 | Lista | Ver la lista con un defecto de cada estado | "Abierto" ofrece **Resolver** y muestra `—`; "Resuelto" ofrece **Reabrir** y muestra "Sprint N" | ✓ |

Verificación hecha el 31/10/2026 con la aplicación en ejecución (`go run ./src/cmd/server`, puerto
8081). Hallazgo: al reabrir sin Sprint activo, el aviso de HU-06 tapaba "Defecto reabierto"; se corrigió
con TDD en T063 y el escenario 3 se volvió a verificar. Al probar el escenario 6, recargar con F5 la
página de error reenvía el formulario: para comprobar que el defecto no cambió hay que volver a
**Defectos** desde el menú.

## Evidencias a guardar en el PR

- `specs/019-defect-resolution/evidencia-red.txt`
- `specs/019-defect-resolution/evidencia-pruebas.txt`
- `specs/019-defect-resolution/cobertura.txt`