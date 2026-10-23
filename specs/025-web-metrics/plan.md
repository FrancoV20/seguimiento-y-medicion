# Implementation Plan: Interfaz web de métricas del Sprint

**Branch**: `HU-25` | **Date**: 2026-10-23 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar la pantalla "Métricas del Sprint" (`/metricas`) como página generada por el servidor y de solo
lectura: lista los Sprints finalizados del proyecto activo y muestra la velocidad y la desviación de esfuerzo del
elegido (por defecto el más reciente). La sección exige proyecto activo, así que HU-20b redirige a la selección
cuando falta. El handler no calcula nada: lee la lista con un catálogo propio (`PostgreSQLMetricsCatalog`), le pide
a HU-07, mediante un adaptador fino (`MetricsSource`), las métricas del Sprint elegido y las presenta con un formato
fijo (desviación con dos decimales, coma y signo, "No disponible" y la alerta de cálculo parcial). No agrega
migraciones y no modifica HU-07, HU-15, HU-17, HU-20a ni HU-20b.

> **Supuestos a verificar antes de implementar (T001, T002 y T004)**:
> 1. La ruta del paquete de HU-07 y la firma de su servicio: entrada (Sprint y proyecto), resultado (velocidad,
>    desviación o su falta, X e Y del cálculo parcial), tipo y redondeo de la desviación, los textos exactos "No
>    disponible" y "Cálculo parcial: basado en X de Y historias" y si exige que el Sprint esté finalizado.
> 2. Cómo calcula HU-07 después de HU-15: que lee las historias "Hecho" del Sprint y no cuenta las devueltas al
>    Backlog; qué devuelve para un Sprint sin historias completadas.
> 3. El valor exacto del estado de Sprint finalizado en `sprints.status` (se supone `'Finalizado'`).
> 4. HU-20a y HU-20b completas en `main`: `Server.Register` con rutas `GET`, `Section.RequiresProject`,
>    `ActiveContextFrom`, `Renderer.Page`, `Fail` y la página 405.
> 5. Los identificadores ya definidos en el paquete `web` (por HU-14, HU-15, HU-21 a HU-24) para que los nombres con
>    prefijo `Metrics` no choquen.

## Technical Context

**Language/Version**: Go 1.27.1

**Primary Dependencies**: Biblioteca estándar de Go (`net/http`, `html/template`, `strconv`) y
`github.com/jackc/pgx/v5/stdlib` ya incorporado. No se agregan dependencias nuevas.

**Storage**: PostgreSQL con Docker Compose. `PostgreSQLMetricsCatalog` solo lee `sprints` mediante `database/sql` y
pgx; el cálculo lo hace el servicio de HU-07 con su propio almacenamiento.

**Testing**: `go test -count=1 -p 1 ./...` con pruebas unitarias del formato, pruebas del adaptador de HU-07 y del
catálogo contra PostgreSQL real, pruebas de handler con `net/http/httptest` y `net/http/cookiejar` (el proyecto
activo se elige por el flujo de HU-20b) y escenarios BDD en `test/web_metrics_bdd_test.go`.

**Target Platform**: Aplicación Go multiplataforma; desarrollo en Windows con Docker Desktop.

**Project Type**: Pantalla de solo lectura de una aplicación web SSR sobre el núcleo de dominio en Go.

**Performance Goals**: La pantalla responde en menos de 500 ms bajo uso normal (SC-005).

**Constraints**: Sin migraciones ni dependencias nuevas, sin JavaScript, cálculos y textos de HU-07 sin copiarse,
ningún detalle interno en las respuestas, y el proyecto saliendo siempre del contexto del servidor.

**Scale/Scope**: Una pantalla y una ruta (`GET`). No incluye Story Points planificados ni porcentaje de historias
completadas (HU-10), detalle de historias (HU-16), gráficos (HU-08) ni exportar (HU-09).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Value-Driven Product Scope**: PASS. Hace visibles las métricas de HU-07, el diferenciador del sistema, y
  cierra la parte visible del MVP; no agrega funcionalidades fuera de la historia.
- **II. Quality by Construction**: PASS. El catálogo, el adaptador de HU-07, el formato y el handler están
  separados; los cálculos siguen en HU-07.
- **III. Test-First and Evidence-Driven Delivery**: PASS. Cada escenario tiene pruebas de handler con datos reales y
  un escenario BDD; las pruebas se escriben antes de implementar.
- **IV. Agile, Collaborative Execution**: PASS. Alcance demostrable en la Sprint Review con un Sprint cerrado.
- **V. Measurement and Continuous Improvement**: PASS. Es la pantalla que mide velocidad y desviación para la
  retrospectiva.
- **VI. PostgreSQL Persistence**: PASS. La interfaz `MetricsCatalog` tiene su adaptador `PostgreSQLMetricsCatalog`
  con prueba de integración contra PostgreSQL; el cálculo reutiliza el almacenamiento PostgreSQL de HU-07.
- **Technology & Quality Constraints**: PASS. Go y PostgreSQL se mantienen; sin dependencias nuevas.

## Project Structure

### Documentation (this feature)

```text
specs/025-web-metrics/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
└── checklists/requirements.md
```

No se crea `contracts/`: la interfaz externa es una página HTML; su comportamiento está en `spec.md` y la dirección
y el campo, abajo.

### Source Code (repository root)

```text
src/web/
├── metrics_catalog.go                # tipos y la interfaz MetricsCatalog
├── postgres_metrics_catalog.go       # adaptador PostgreSQL (solo lectura)
├── metrics_source.go                 # interfaz MetricsSource y adaptador sobre el servicio de HU-07
├── metrics_view.go                   # textos, formato, selección del Sprint y armado de la vista
├── metrics_handlers.go               # GET /metricas
├── templates/metrics.html            # lista de Sprints y métricas del elegido
└── *_test.go
test/
└── web_metrics_bdd_test.go
```

**Structure Decision**: Todo lo nuevo vive en `src/web` con el prefijo `Metrics` y no edita archivos de otras
historias. Se edita `src/cmd/server/main.go` para construir el catálogo y el adaptador y registrar la sección.

### Dirección y campo (propuestos)

| Método | Dirección | Efecto |
| --- | --- | --- |
| GET | `/metricas` | Lista de Sprints finalizados y métricas del más reciente |
| GET | `/metricas?sprint=ID` | Igual, con métricas del Sprint elegido |

La sección se registra como `Section{Name: "Métricas del Sprint", Path: "/metricas", RequiresProject: true}`. No
requiere integrante. El proyecto sale del servidor.

### Flujo del GET

1. Si no hay proyecto activo válido, HU-20b redirige con 303 a la selección de proyecto.
2. Leer los Sprints finalizados del proyecto (más reciente primero) y el número del Sprint activo, si hay.
3. Elegir el Sprint: el de `sprint` si es numérico y está en la lista; si no, el primero de la lista. Sin lista, no
   hay selección.
4. Con selección, pedir a HU-07 las métricas de ese Sprint y armar la vista (velocidad, desviación, alerta).
5. Responder 200 con la página. Si falla la lectura o el cálculo, usar `Fail` de HU-20a.

### Formato de la vista

| Dato | Formato |
| --- | --- |
| Velocidad | `N Story Points` (cero se muestra como `0 Story Points`) |
| Desviación con datos | dos decimales, coma y signo: `+20,00 %`, `-8,50 %`, `0,00 %` |
| Desviación sin horas estimadas | "No disponible" |
| Cálculo parcial | "Cálculo parcial: basado en X de Y historias" |
| Fechas | las del formato de fecha del resto de las pantallas |

### Contratos que consume

- De HU-20a: `Server.Register(section, routes)`, `Renderer.Page`, `Fail` y `RenderError` (y la página 405).
- De HU-20b: `ActiveContextFrom` y la exigencia de proyecto por sección.
- De HU-07: el servicio que calcula velocidad y desviación y sus textos.
- De HU-15: Sprint "Finalizado" con las historias "Hecho" asignadas.

## Complexity Tracking

> Sin violaciones de la constitución que justificar.

## Post-Design Constitution Check

- **I. Value-Driven Product Scope**: PASS. El diseño se limita a mostrar lo que calcula HU-07.
- **II. Quality by Construction**: PASS. Catálogo, adaptador, formato y handler separados; cálculos en HU-07.
- **III. Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas unitarias, de integración,
  de handler y BDD.
- **IV. Agile, Collaborative Execution**: PASS. Las dos historias de usuario se entregan de forma incremental.
- **V. Measurement and Continuous Improvement**: PASS. Sin cambios en los cálculos.
- **VI. PostgreSQL Persistence**: PASS. Catálogo con adaptador y prueba de integración contra PostgreSQL.
- **Technology & Quality Constraints**: PASS. Sin dependencias ni tecnologías nuevas.