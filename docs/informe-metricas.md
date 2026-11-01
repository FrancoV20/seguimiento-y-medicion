# Informe de métricas y cobertura

Corte: **1 de noviembre de 2026**, después de integrar HU-08 (PR #68) al cierre del Sprint 3.

## Resumen

| Indicador | Valor |
|---|---|
| Cobertura total (CI sobre `main`) | **86,1 %** |
| Paquetes de reglas de negocio y cálculos con ≥ 80 % (DoD) | **7 de 7** (mínimo: `sprint`, 83,3 %) |
| Funciones de prueba (`func Test…`) | 1089, en 184 archivos `_test.go` |
| Pruebas de escenarios BDD de punta a punta (`test/`) | 217, en 29 archivos |
| Líneas de Go: código de producción / código de pruebas | 14.055 / 44.899 |
| Estado del pipeline de CI en `main` | Verde en los 31 runs registrados |
| Velocidad acumulada (Sprints 1 a 3) | 125 SP, con un promedio de 41,67 SP por Sprint |

## Cobertura por paquete (DoD: mínimo 80 % en reglas de negocio y cálculos)

Fuente: run del workflow CI sobre `main` en el merge del PR #68 (01/11/2026, 19:15), paso
`go test ./... -p 1 -race -coverprofile=coverage.out -covermode=atomic`.

| Paquete | Contenido | Cobertura | ≥ 80 % |
|---|---|---|---|
| `src/backlog` | Product Backlog: alta, edición y validaciones de historias | 89,5 % | Sí |
| `src/defect` | Registro, resolución y reapertura de defectos | 87,8 % | Sí |
| `src/effort` | Registro de esfuerzo y validación de horas | 92,5 % | Sí |
| `src/metrics` | Velocidad, desviación, calidad y avance; validación de cierre | 85,9 % | Sí |
| `src/poker` | Planning Poker: votos, revelación, divergencia y consenso | 88,5 % | Sí |
| `src/project` | Proyectos, integrantes, fechas y estado | 87,3 % | Sí |
| `src/sprint` | Sprints, asignación, estado de historias y cierre | 83,3 % | Sí |
| `src/web` | Pantallas, formularios, contexto de trabajo y errores | 90,0 % | – |
| `db` | Migraciones | 83,3 % | – |
| `src/cmd/server` | Arranque del servidor | 5,9 % | – |

`src/cmd/server` solo contiene el arranque (lectura de configuración, conexión y apagado), que se
prueba con la aplicación en ejecución y no forma parte de las reglas de negocio. `src/web` y `db`
no son reglas de negocio, pero también superan el 80 %.

## Evolución de la cobertura total

Cobertura total que informa el paso "Resumen de cobertura" del CI (`go tool cover -func`) en cada
integración a `main`:

| Fecha | Integración | Sprint | Cobertura total |
|---|---|---|---|
| 27/10/2026 | chore/ci (#44), primer run del pipeline | 2 | 85,3 % |
| 27/10/2026 | HU-24 (#37) | 2 | 85,8 % |
| 28/10/2026 | HU-26 (#42) | 3 | 85,9 % |
| 28/10/2026 | HU-05a (#46) | 3 | 85,9 % |
| 28/10/2026 | HU-23 (#50) | 2 | 84,9 % |
| 28/10/2026 | HU-06 (#48) | 3 | 84,9 % |
| 29/10/2026 | FIX-HU-21 (#58) | 3 | 84,9 % |
| 29/10/2026 | HU-11 (#54) | 3 | 85,2 % |
| 29/10/2026 | HU-05b (#55) | 3 | 85,4 % |
| 29/10/2026 | HU-15 (#59) | 2 | 85,4 % |
| 30/10/2026 | HU-17 (#56) | 2 | 85,4 % |
| 31/10/2026 | HU-12 (#60) | 3 | 85,4 % |
| 31/10/2026 | HU-18 (#61) | 3 | 85,5 % |
| 31/10/2026 | HU-19 (#62) | 3 | 85,5 % |
| 01/11/2026 | HU-25 (#64) | 2 | 85,6 % |
| 01/11/2026 | HU-16 (#65) | 3 | 85,7 % |
| 01/11/2026 | HU-10 (#66) | 3 | 85,7 % |
| 01/11/2026 | HU-13 (#67) | 4 | 85,8 % |
| 01/11/2026 | HU-08 (#68) | 3 | **86,1 %** |

La cobertura total se mantuvo entre 84,9 % y 86,1 % mientras el código crecía en 19 integraciones.
Cada historia llegó con sus pruebas (TDD) y ninguna bajó la cobertura por debajo del 80 %.

Observación: según la evidencia de HU-20b (`specs/020b-project-context/cobertura.txt`), al
integrar esa historia el paquete `sprint` quedó en 79,4 %. En las integraciones siguientes volvió
a superar el 80 %, y desde entonces se mantiene en 80,7 % o más.

## Métricas del proyecto por Sprint

| Sprint | SP planificados | SP completados (velocidad) | Historias completadas | Defectos detectados | Defectos resueltos |
|---|---|---|---|---|---|
| Sprint 1 | 21 (+3 de HU-03, incorporada) | 24 | 5 de 5 (100,00 %) | 0 | 0 |
| Sprint 2 | 44 | 44 | 10 de 10 (100,00 %) | 2 (FIX-HU-01, FIX-HU-04) | 2 |
| Sprint 3 | 57 | 57 | 11 de 11 (100,00 %) | 1 (FIX-HU-21) | 1 |
| Sprint 4 | 19 comprometidos + 13 extra | (al cierre) | (al cierre) | (al cierre) | (al cierre) |

- **Velocidad:** 24 → 44 → 57 SP. Acumulada: 125 SP; promedio: 41,67 SP por Sprint.
- **Defectos:** se registraron 3, todos resueltos dentro del mismo Sprint en que se detectaron.
  Hoy no hay defectos abiertos. FIX-HU-01 y FIX-HU-04 corrigieron reglas de historias del
  Sprint 1. FIX-HU-21 corrigió una incompatibilidad entre HU-21 y HU-22 (estado "Active" contra
  "Activo") que las pruebas no detectaban, porque creaban los proyectos directo en la base.
- **Precisión de la estimación:** entre la estimación inicial (Planning Poker del 25/09) y la del
  31/10, HU-06, HU-10 y HU-09 pasaron de 5 a 8 SP, porque el refinamiento les agregó escenarios.
- **Horas estimadas, horas reales y desviación del equipo:** (completar si el equipo registró sus
  horas; se pueden calcular cargando los Sprints en la propia aplicación, en "Métricas del Sprint").

## Cómo reproducir

```bash
docker compose up -d
go test ./... -count=1 -p 1 -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out | tail -n 1     # cobertura total
go tool cover -html=coverage.out                 # detalle por línea
```

El workflow CI (`.github/workflows/ci.yml`) ejecuta lo mismo en cada push y PR a `main`, y guarda
`coverage.out` como artefacto `coverage` del run. La evidencia de cada historia está en
`specs/NNN-nombre/cobertura.txt`, `evidencia-red.txt` (fase RED) y `evidencia-pruebas.txt`
(fase GREEN).
