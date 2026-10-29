# Plan de implementación: HU-10 – Cálculo de Métricas de Calidad y Avance

**Rama**: `HU-10` | **Fecha**: 2026-10-28 | **Spec**: [spec.md](spec.md)
**Entrada**: `specs/010-quality-metrics/spec.md`
**Orden de implementación**: después de HU-06 y HU-15, que tienen que estar en `main` antes de crear la rama (T001).

## Resumen

HU-10 calcula, para un Sprint finalizado, el avance (historias planificadas, completadas y porcentaje), los Story
Points planificados y completados, y los defectos detectados y resueltos. Los muestra en la sección "Calidad y
avance" (`GET /calidad?sprint=ID`), de solo lectura.

- El cálculo vive en `src/metrics`, en archivos nuevos: el puerto `QualityStore`, su adaptador PostgreSQL y el
  servicio `QualityService.SprintQuality`.
- Reutiliza el redondeo y el error de Sprint no finalizado de HU-07.
- Lee el registro de cierre de HU-15 y los defectos de HU-06 en una sola transacción de solo lectura.
- La pantalla usa un catálogo propio en `src/web`, con el prefijo `Quality`.
- No hay migración y no se guarda nada.

## Contexto técnico

**Lenguaje/Versión**: Go 1.27.1
**Dependencias**: biblioteca estándar (`net/http`, `html/template`, `database/sql`, `math/big`) y
`github.com/jackc/pgx/v5/stdlib`, que ya están en el proyecto. No se agregan dependencias.
**Almacenamiento**: PostgreSQL. Sin migración: solo lectura de `sprints`, `sprint_story_results` (HU-15) y `defects`
(HU-06).
**Pruebas**:

- unitarias del cálculo y del servicio, con un almacén de prueba local;
- integración contra PostgreSQL real con datos propios;
- handlers con `httptest`;
- BDD en `test/`.

Se corren con `go test -count=1 -p 1 ./...`.
**CI**: `gofmt -l` sin salida, `go vet ./...`, `go test ./... -p 1 -race` con cobertura. Se formatean solo los
archivos que se crean o se tocan.
**Plataforma**: servidor web local (`src/cmd/server`) en Windows y Linux (CI).
**Tipo de proyecto**: aplicación web renderizada en el servidor, sin JavaScript.
**Objetivos de rendimiento**: menos de 500 ms con 20 Sprints, 50 historias por Sprint y 100 defectos (SC-005).
**Restricciones**:

- No se editan archivos de HU-07 ni de HU-25.
- El único archivo compartido que se edita es `src/cmd/server/main.go`.
- Ningún mensaje expone detalles internos.

**Escala**: proyectos del curso.

## Verificación de la constitución

| Principio | Cómo se cumple | Estado |
|---|---|---|
| I. Alcance guiado por valor | Should Have, valor 75. Cada FR sale de un escenario del Issue o de una decisión del equipo. | PASA |
| II. Calidad por construcción | El cálculo es puro y está separado de la lectura. Se reutiliza el redondeo de HU-07, sin copiarlo. Los textos están en un solo lugar. | PASA |
| III. Primero las pruebas | Escenarios en Dado/Cuando/Entonces. Cada bloque sigue RED → `evidencia-red.txt` → GREEN → REFACTOR. | PASA |
| IV. Trabajo ágil y colaborativo | T001 confirma los nombres de HU-06 y HU-15. T004 y T038 coordinan el bloque de `main.go` y avisan a HU-08 y HU-09 del contrato. | PASA |
| V. Medición | Agrega las métricas de avance y calidad que pide la constitución ("defect trends", "sprint progress") y las cruza con la velocidad de HU-07. | PASA |
| VI. Persistencia PostgreSQL | `QualityStore` y `QualityCatalog` tienen adaptador PostgreSQL y pruebas de integración. | PASA |

## Estructura

### Documentación

```text
specs/010-quality-metrics/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
├── checklists/requirements.md
├── evidencia-red.txt        (durante la implementación)
├── evidencia-pruebas.txt    (cierre)
└── cobertura.txt            (cierre)
```

No hay carpeta `contracts/`. El contrato para HU-08 y HU-09 (`QualityService.SprintQuality` y `SprintQuality`) está
en [data-model.md](data-model.md), y la ruta, en este plan.

### Código

Archivos nuevos:

```text
src/metrics/quality.go                             tipos, ErrSprintNotFound, calculateSprintQuality
src/metrics/quality_store.go                       puerto QualityStore
src/metrics/quality_service.go                     QualityService, NewQualityService, SprintQuality
src/metrics/postgres_quality_store.go              adaptador PostgreSQL (solo lectura)
src/metrics/quality_test.go                        cálculo puro
src/metrics/quality_service_test.go                servicio con un almacén de prueba local
src/metrics/postgres_quality_store_integration_test.go
src/web/quality_catalog.go                         QualityCatalog, QualityReader y tipos
src/web/postgres_quality_catalog.go                adaptador PostgreSQL de la lista de Sprints
src/web/quality_view.go                            textos, formato y elección del Sprint
src/web/quality_handlers.go                        RegisterQuality y GET /calidad
src/web/templates/quality.html
src/web/postgres_quality_catalog_integration_test.go
src/web/quality_view_test.go
src/web/quality_handlers_test.go
test/web_quality_bdd_test.go
```

Archivo existente que se edita: `src/cmd/server/main.go`, para sumar un bloque de registro después del último (hoy
el de HU-24).

No se tocan `src/metrics/{calculation,closure_validation,errors,model,service,store,test_store,postgres_metrics_store}.go`
(HU-07), los archivos de HU-25, `db/migrations` ni los de HU-06 y HU-15.

## Rutas

| Método y ruta | Handler | Respuestas |
|---|---|---|
| `GET /calidad[?sprint=ID]` | `get` | 200 (lista y datos, o "Aún no hay Sprints finalizados"); 303 sin proyecto; 500 o 503 si falla la lectura |
| `POST /calidad` (u otro método) | — | 405 de HU-20a |

## Decisiones de diseño

Detalle y alternativas en [research.md](research.md).

1. Puerto nuevo `QualityStore`, con `context`. No se extiende `MetricsStore`.
2. El cálculo puro `calculateSprintQuality` usa `roundToTwoDecimals` de HU-07. El porcentaje es texto con punto, o
   "No disponible".
3. La lectura es de solo lectura con `REPEATABLE READ`: Sprint del proyecto, registro de cierre y los dos conteos
   de defectos con `COUNT(*) FILTER`.
4. Un identificador inválido, o un Sprint inexistente o ajeno, da `ErrSprintNotFound` (nuevo). Un Sprint activo da
   `ErrSprintNotFinished` (de HU-07).
5. Una prueba de coherencia compara `PointsCompleted` con la velocidad de HU-07.
6. La web usa el catálogo `QualityCatalog` para la lista y `QualityReader` (que implementa el servicio) para los
   números. El prefijo `Quality` evita chocar con HU-25.
7. Un `sprint` que no está entre los finalizados se ignora. Cualquier falla va a `Fail`, sin números parciales.
8. Los textos están en `quality_view.go`.

## Contratos que se consumen

- **HU-07**: `roundToTwoDecimals`, `ErrSprintNotFinished`, `NewService`, `NewPostgreSQLMetricsStore` y
  `CalculateMetrics` (solo en la prueba de coherencia).
- **HU-15**: `sprint_story_results(sprint_id, story_id, final_status, story_points)` **(T001)**.
- **HU-06 y HU-19**: `defects(project_id, status, detection_sprint_id, resolution_sprint_id)` **(T001)**.
- **HU-20a**: `Server.Register`, `Routes`, `Renderer.PageStatus/Fail`, `Unavailable`, el menú, `layout.html` y el
  405 de `Server.route`.
- **HU-20b**: `Section.RequiresProject`, `RequireContext`, `ActiveContextFrom` y `parseID`.

## Contrato que se ofrece (HU-08 y HU-09)

```go
svc := metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(db))
q, err := svc.SprintQuality(ctx, projectID, sprintID) // metrics.SprintQuality
```

## Riesgos

| Riesgo | Mitigación |
|---|---|
| HU-06 y HU-15 no están en `main`, y sus nombres pueden cambiar al implementarse | T001 los confirma con `git grep -n` antes de crear la rama y ajusta `data-model.md` y las consultas. Sin las dos en `main` no se empieza. |
| Sprints "Finalizado" sin registro de cierre (datos viejos o cargados a mano) muestran "No disponible" aunque hayan tenido historias | Está documentado en los casos límite de la spec, y una prueba lo fija como comportamiento esperado. |
| La velocidad de HU-07 usa los Story Points actuales, y HU-10 usa los del cierre | Coinciden mientras no se editen las historias de un Sprint finalizado (lo bloquea HU-13). La prueba de coherencia lo verifica sobre el mismo Sprint. |
| `CalculateMetrics` de HU-07 guarda en `sprint_metrics` durante la prueba de coherencia | La prueba borra sus filas en la limpieza. HU-10 no lo llama fuera de esa prueba. |
| Conflicto en `main.go` con HU-08, HU-09, HU-25 u otras historias | Un bloque independiente; al mergear se conservan todos los bloques (T004, T038). |
| El `TRUNCATE` del CI pisa datos de otras pruebas | Cada prueba arma sus datos y corre con `-p 1`. |

## Verificación posterior al diseño

Vuelvo a revisar el diseño cerrado contra la constitución:

- **Principio VI**: los dos puertos nuevos tienen adaptador PostgreSQL y prueba de integración.
- **Principio II**: no se duplicó el redondeo ni los textos.
- **Principio III**: las tareas siguen RED → evidencia → GREEN en cada bloque.

Sigue en **PASA**, sin violaciones que justificar.
