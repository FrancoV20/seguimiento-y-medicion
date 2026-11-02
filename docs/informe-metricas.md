# Informe de métricas y cobertura

Corte: **2 de noviembre de 2026**, después de integrar MEJORA-01 (PR #72), al cierre del Sprint 4
y antes de la Sprint Review final.

## Resumen

| Indicador | Valor |
|---|---|
| Cobertura total (CI sobre `main`) | **87,2 %** |
| Paquetes de reglas de negocio y cálculos con ≥ 80 % (DoD) | **11 de 11** (mínimo: `sprint`, 83,3 %) |
| Funciones de prueba (`func Test…`) | 1218, en 205 archivos `_test.go` (223 archivos de prueba en total, contando los auxiliares) |
| Pruebas de escenarios BDD de punta a punta (`test/`) | 249, en 33 archivos |
| Líneas de Go: código de producción / código de pruebas | 17.103 / 52.269 |
| Estado del pipeline de CI en `main` | Verde en los 36 runs registrados |
| Velocidad acumulada (Sprints 1 a 4) | 157 SP, con un promedio de 39,25 SP por Sprint |
| Requisitos mínimos del enunciado cubiertos | 9 de 9 |

## Cobertura por paquete (DoD: mínimo 80 % en reglas de negocio y cálculos)

Fuente: run del workflow CI sobre `main` en el merge del PR #72 (02/11/2026, 14:05), paso
`go test ./... -p 1 -race -coverprofile=coverage.out -covermode=atomic`.

### Reglas de negocio y cálculos

| Paquete | Contenido | Cobertura | ≥ 80 % |
|---|---|---|---|
| `src/auth` | Contraseñas (PBKDF2-SHA256) y bloqueo por intentos fallidos | 94,9 % | Sí |
| `src/backlog` | Product Backlog: alta, edición, horas estimadas y validaciones de historias | 89,5 % | Sí |
| `src/defect` | Registro, resolución y reapertura de defectos | 87,8 % | Sí |
| `src/effort` | Registro de esfuerzo y validación de horas | 92,5 % | Sí |
| `src/metrics` | Velocidad, desviación, calidad y avance; validación de cierre | 85,9 % | Sí |
| `src/pdf` | Generador de PDF propio: texto, tablas, paginado y pie "Página X de Y" | 90,3 % | Sí |
| `src/poker` | Planning Poker: votos, revelación, divergencia y consenso | 88,5 % | Sí |
| `src/project` | Proyectos, integrantes, fechas y estado | 87,3 % | Sí |
| `src/report` | Contenido de los reportes de Sprint y de proyecto | 90,9 % | Sí |
| `src/role` | Roles del proyecto (Scrum Master e Integrante) | 100,0 % | Sí |
| `src/sprint` | Sprints, asignación, estado de historias y cierre | 83,3 % | Sí |

### Resto del código

| Paquete | Contenido | Cobertura |
|---|---|---|
| `src/web` | Pantallas, formularios, contexto de trabajo, permisos y errores | 90,6 % |
| `src/cmd/set-password` | Comando para configurar la contraseña de un integrante | 88,9 % |
| `src/cmd/set-role` | Comando para asignar el rol de un integrante | 94,5 % |
| `db` | Migraciones | 83,3 % |
| `src/cmd/server` | Arranque del servidor | 5,7 % |

`src/cmd/server` solo contiene el arranque (lectura de configuración, conexión, verificación del
esquema y apagado), que se prueba con la aplicación en ejecución y no forma parte de las reglas de
negocio. El resto del código, aunque no son reglas de negocio, también supera el 80 %.

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
| 01/11/2026 | HU-08 (#68) | 3 | 86,1 % |
| 01/11/2026 | HU-09 (#69) | 4 | 86,7 % |
| 02/11/2026 | HU-27 (#70) | 4 | 86,9 % |
| 02/11/2026 | HU-28 (#71) | 4 | 87,2 % |
| 02/11/2026 | MEJORA-01 (#72) | 4 | **87,2 %** |

La cobertura total se mantuvo entre 84,9 % y 87,2 % mientras el código crecía en 23 integraciones,
y en el Sprint 4 subió en cada merge. Cada historia llegó con sus pruebas (TDD) y ninguna bajó un
paquete de reglas de negocio por debajo del 80 %.

Observación: según la evidencia de HU-20b (`specs/020b-project-context/cobertura.txt`), al
integrar esa historia el paquete `sprint` quedó en 79,4 %. En las integraciones siguientes volvió
a superar el 80 %, y desde entonces se mantiene en 80,7 % o más (hoy, 83,3 %).

## Métricas del proyecto por Sprint

| Sprint | SP planificados | SP completados (velocidad) | Historias completadas | Defectos detectados | Defectos resueltos |
|---|---|---|---|---|---|
| Sprint 1 | 21 (+3 de HU-03, incorporada) | 24 | 5 de 5 (100,00 %) | 0 | 0 |
| Sprint 2 | 44 | 44 | 10 de 10 (100,00 %) | 2 (FIX-HU-01, FIX-HU-04) | 2 |
| Sprint 3 | 57 | 57 | 11 de 11 (100,00 %) | 1 (FIX-HU-21) | 1 |
| Sprint 4 | 32 (19 comprometidos + 13 de objetivo extra) | 32 | 5 de 5 (100,00 %) | 0 | 0 |
| **Total** | **157** | **157** | **31 de 31 (100,00 %)** | **3** | **3** |

- **Velocidad:** 24 → 44 → 57 → 32 SP. Acumulada: 157 SP; promedio: 39,25 SP por Sprint. El
  Sprint 4 duró dos días (1 y 2 de noviembre), por eso su velocidad es menor: por día fue el
  Sprint más productivo (16 SP por día).
- **Sprint 4:** se completaron las tres historias comprometidas (HU-13, HU-09 y MEJORA-01, 19 SP)
  y también las dos del objetivo extra (HU-27 y HU-28, 13 SP).
- **Defectos:** se registraron 3 en todo el proyecto, todos resueltos dentro del mismo Sprint en
  que se detectaron. Hoy no hay defectos abiertos. FIX-HU-01 y FIX-HU-04 corrigieron reglas de
  historias del Sprint 1. FIX-HU-21 corrigió una incompatibilidad entre HU-21 y HU-22 (estado
  "Active" contra "Activo") que las pruebas no detectaban, porque creaban los proyectos directo en
  la base. En el Sprint 4, el problema de los PDF de evidencia de HU-09 (finales de línea
  convertidos por Git en Windows) se detectó en la revisión del PR, antes del merge, y se corrigió
  con `.gitattributes`; no llegó a `main`, por eso no se cuenta como defecto.
- **Precisión de la estimación:** entre la estimación inicial (Planning Poker del 25/09) y la del
  31/10, HU-06, HU-10 y HU-09 pasaron de 5 a 8 SP, porque el refinamiento les agregó escenarios.
- **Horas estimadas, horas reales y desviación:** la aplicación las calcula por Sprint (HU-07,
  "Métricas del Sprint") y las incluye en el Dashboard y en los reportes PDF. El equipo no cargó
  en la aplicación las horas de su propio trabajo durante el desarrollo, así que este informe no
  tiene una desviación de esfuerzo del proyecto. El cálculo se demuestra con los datos de prueba
  de la Sprint Review y está cubierto por las pruebas de `src/metrics` (85,9 %).

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
