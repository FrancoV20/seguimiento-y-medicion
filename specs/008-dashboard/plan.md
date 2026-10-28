# Plan de implementación: HU-08 – Visualización de Dashboard del Proyecto

**Rama**: `HU-08` | **Fecha**: 2026-10-28 | **Spec**: [spec.md](spec.md)
**Entrada**: `specs/008-dashboard/spec.md`
**Orden de implementación**: después de HU-10 (y por lo tanto de HU-06 y HU-15), que tiene que estar en `main` antes
de crear la rama (T001).

## Resumen

HU-08 agrega la sección "Dashboard" (`GET /dashboard`), de solo lectura, con cinco bloques del proyecto activo:
historias por estado, Sprint en curso, velocidad del equipo, desviación del último Sprint finalizado y defectos
abiertos. Los dos gráficos son barras SVG armadas en el servidor, cada una con su tabla.

- No recalcula lo que ya existe: el Sprint en curso es el Tablero de HU-14, la velocidad por Sprint es la de HU-10 y
  la desviación es la de HU-07.
- Lo nuevo del dominio es `metrics.SummarizeVelocity` (acumulada y promedio), en un archivo nuevo de `src/metrics`.
- Lo nuevo de lectura es `DashboardCatalog`: historias por estado, defectos abiertos y Sprints finalizados, en una
  sola transacción de solo lectura.
- No hay migración y no se guarda nada propio.

## Contexto técnico

**Lenguaje/Versión**: Go 1.27.1
**Dependencias**: biblioteca estándar (`net/http`, `html/template`, `database/sql`, `math/big`) y
`github.com/jackc/pgx/v5/stdlib`, que ya están en el proyecto. No se agregan dependencias.
**Almacenamiento**: PostgreSQL. Sin migración: lectura de `user_stories`, `sprints` y `defects` (HU-06); HU-10 y
HU-07 leen lo suyo, y HU-07 reescribe `sprint_metrics` del último Sprint con los mismos valores.
**Pruebas**:

- unitarias de `SummarizeVelocity`, de las alturas de barras y de los formatos;
- integración del catálogo contra PostgreSQL real con datos propios;
- handlers con `httptest`, con los servicios reales y datos propios, y con dobles que fallan para los errores;
- BDD en `test/`.

Se corren con `go test -count=1 -p 1 ./...`. Limpieza de los datos de prueba: defectos, registro de cierre,
asignaciones, `sprint_metrics`, esfuerzo, historias, Sprints, proyectos e integrante (research.md §13).
**CI**: `gofmt -l` sin salida, `go vet ./...`, `go test ./... -p 1 -race` con cobertura. Se formatean solo los
archivos que se crean o se tocan.
**Plataforma**: servidor web local (`src/cmd/server`) en Windows y Linux (CI).
**Tipo de proyecto**: aplicación web renderizada en el servidor, sin JavaScript.
**Objetivos de rendimiento**: menos de 500 ms con 20 Sprints finalizados, 200 historias y 100 defectos (SC-005).
**Restricciones**:

- No se editan archivos de HU-07, HU-10, HU-14 ni HU-25.
- El único archivo compartido que se edita es `src/cmd/server/main.go`.
- Ningún mensaje expone detalles internos.

**Escala**: proyectos del curso.

## Verificación de la constitución

| Principio | Cómo se cumple | Estado |
|---|---|---|
| I. Alcance guiado por valor | Should Have, valor 75. Cada FR sale de un escenario del Issue o de una decisión del equipo; el Dashboard es una de las capacidades que pide la constitución ("measurement dashboards"). | PASA |
| II. Calidad por construcción | Se reutilizan el Tablero (HU-14), `SprintQuality` (HU-10), `CalculateMetrics` y `roundToTwoDecimals` (HU-07) y `QualityReader`, sin copiarlos. La regla de promedio está en el dominio. Los textos están en un solo lugar. | PASA |
| III. Primero las pruebas | Escenarios en Dado/Cuando/Entonces. Cada bloque sigue RED → `evidencia-red.txt` → GREEN → REFACTOR. | PASA |
| IV. Trabajo ágil y colaborativo | T001 confirma los nombres de HU-06, HU-07, HU-10, HU-14, HU-15 y HU-25 y frena si HU-10 no está. T004 y T035 coordinan `main.go` y avisan a HU-09 y HU-25. | PASA |
| V. Medición | Reúne en una pantalla velocidad, desviación de esfuerzo, avance del Sprint y defectos ("project health indicators"). | PASA |
| VI. Persistencia PostgreSQL | El único puerto nuevo, `DashboardCatalog`, tiene adaptador PostgreSQL y prueba de integración. Las interfaces reutilizadas ya tienen el suyo. `DashboardDeviationSource` no es un puerto de almacenamiento: es la forma de `*metrics.Service`. | PASA |

## Estructura

### Documentación

```text
specs/008-dashboard/
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

No hay carpeta `contracts/`: la única interfaz externa es la ruta `GET /dashboard` (tabla de rutas de este plan) y
los tipos están en [data-model.md](data-model.md).

### Código

Archivos nuevos:

```text
src/metrics/velocity_summary.go                      VelocitySummary, SummarizeVelocity
src/metrics/velocity_summary_test.go
src/web/dashboard_catalog.go                         tipos, DashboardCatalog, DashboardDeviationSource
src/web/postgres_dashboard_catalog.go                adaptador PostgreSQL (solo lectura)
src/web/postgres_dashboard_catalog_integration_test.go
src/web/dashboard_chart.go                           dashboardBarHeights y armado del SVG
src/web/dashboard_chart_test.go
src/web/dashboard_view.go                            textos, formatos y buildDashboardView
src/web/dashboard_view_test.go
src/web/dashboard_handlers.go                        RegisterDashboard y GET /dashboard
src/web/dashboard_handlers_test.go
src/web/templates/dashboard.html
test/web_dashboard_bdd_test.go
```

Archivo existente que se edita: `src/cmd/server/main.go`, para sumar un bloque de registro después del último.

No se tocan los archivos de HU-07 (`src/metrics/{calculation,closure_validation,errors,model,service,store,
test_store,postgres_metrics_store}.go`), de HU-10 (`src/metrics/quality*.go`, `src/web/quality*.go`), de HU-14
(`src/web/board*.go`, `postgres_board_catalog.go`), de HU-25 (`src/web/metrics*.go`) ni `db/migrations`.

## Rutas

| Método y ruta | Handler | Respuestas |
|---|---|---|
| `GET /dashboard` | `show` | 200 (cinco bloques, o el mensaje de proyecto sin datos); 303 sin proyecto; 500 o 503 si falla una lectura o un cálculo |
| `POST /dashboard` (u otro método) | — | 405 de HU-20a |

## Decisiones de diseño

Detalle y alternativas en [research.md](research.md).

1. `SummarizeVelocity` en `src/metrics/velocity_summary.go`, con `roundToTwoDecimals` de HU-07; devuelve texto con
   punto, y la coma se pone en la pantalla.
2. Se reutiliza `QualityReader` de HU-10 **(T001)**; para la desviación, una interfaz chica que `*metrics.Service`
   cumple sin adaptador.
3. `DashboardCatalog` lee historias por estado, defectos abiertos y Sprints finalizados en una transacción
   `ReadOnly` + `REPEATABLE READ`.
4. Orden: catálogo → Tablero → (si no hay datos, mensaje único) → HU-10 por Sprint → HU-07 del último. Cualquier error
   va a `Fail` antes de escribir la respuesta.
5. Barras SVG con `html/template`, `role="img"`, `aria-label` y tabla; las alturas, con `dashboardBarHeights`.
6. La desviación se formatea con la función de HU-25 si está en `main`, o con `dashboardDeviationText` **(T001)**;
   el aviso parcial de HU-07 se muestra tal cual.
7. Los textos están en `dashboard_view.go`.

## Contratos que se consumen

- **HU-07**: `NewService`, `NewPostgreSQLMetricsStore`, `CalculateMetrics` (`DeviationPercentage`, `IsPartial`,
  `Warning`) y `roundToTwoDecimals`.
- **HU-10** **(T001)**: `NewQualityService`, `NewPostgreSQLQualityStore`, `SprintQuality(...).PointsCompleted` y
  `web.QualityReader`.
- **HU-14**: `BoardCatalog.ActiveBoard`, `BoardView` (`Number`, `Groups`, `PlannedPoints`, `CompletedPoints`),
  `NewPostgreSQLBoardCatalog` y `projectIDFrom`.
- **HU-06** **(T001)**: `defects(project_id, status)`.
- **HU-15** **(T001)**: `sprint_story_results`, solo en los datos de prueba.
- **HU-25** **(T001)**: la función de formato de la desviación, si está en `main`.
- **HU-20a / HU-20b**: `Server.Register`, `Routes`, `Renderer.Page/Fail`, el menú, `layout.html`, el 405 de
  `Server.route`, `Section.RequiresProject` y `RequireContext`.

## Riesgos

| Riesgo | Mitigación |
|---|---|
| HU-10 (y con ella HU-06 y HU-15) no está en `main`, o sus nombres cambian | T001 los confirma con `git grep -n` antes de crear la rama, ajusta `data-model.md` y frena si HU-10 no está. |
| `CalculateMetrics` reescribe `sprint_metrics` en cada apertura del Dashboard | Escribe los mismos valores, como en HU-25 (decisión aceptada). Las pruebas borran esas filas. |
| HU-07 rechaza el último Sprint (historia "Hecho" sin Story Points cargada a mano) y el Dashboard entero muestra la página de error | Es el comportamiento pedido (cualquier falla → error); queda como caso límite con prueba. Con el cierre de HU-15 no se puede llegar a ese estado. |
| Un Sprint finalizado sin registro de cierre de HU-15 muestra velocidad 0 aunque HU-07 calcule otra | Caso límite documentado; la prueba de coherencia usa Sprints cerrados como los deja HU-15. |
| Una llamada a HU-10 por Sprint finalizado (N + 1) | Medido en `TestHU08_Rendimiento` con 20 Sprints; menos de 500 ms. |
| `html/template` dentro de `<svg>` | Solo se interpolan números y textos fijos o escapados; una prueba verifica el SVG generado. |
| Conflicto en `main.go` con HU-09, HU-10, HU-25 u otras historias | Un bloque independiente al final; al mergear se conservan todos (T004, T035). |
| El `TRUNCATE` del CI o pruebas en paralelo pisan datos | Cada prueba arma sus datos, los borra al final y corre con `-p 1`. |

## Verificación posterior al diseño

Vuelvo a revisar el diseño cerrado contra la constitución:

- **Principio VI**: `DashboardCatalog` tiene adaptador PostgreSQL y prueba de integración.
- **Principio II**: no se duplicaron el redondeo, el cálculo de velocidad, el de desviación ni el Tablero.
- **Principio III**: las tareas siguen RED → evidencia → GREEN en cada bloque.

Sigue en **PASA**, sin violaciones que justificar.
