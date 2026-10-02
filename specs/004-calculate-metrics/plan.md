# Implementation Plan: Calcular métricas del Sprint

**Branch**: `004-calculate-metrics` | **Date**: 2026-10-01 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar un servicio de dominio Go que calcule de forma determinista la velocidad y la
desviación de esfuerzo de un Sprint finalizado. El cálculo usará únicamente historias
terminadas con horas estimadas y reales completas, informará el conteo de un cálculo parcial,
devolverá `No disponible` cuando el denominador sea cero y validará los datos inválidos antes
de permitir el cierre del Sprint. `PostgreSQLMetricsStore` leerá las fuentes y persistirá el
resultado mediante pgx a través de `database/sql`.

## Technical Context

**Language/Version**: Go 1.22 o posterior, fijado en `go.mod` durante la inicialización

**Primary Dependencies**: Biblioteca estándar de Go, `github.com/jackc/pgx/v5` y
`github.com/jackc/pgx/v5/stdlib` para registrar pgx mediante `database/sql`

**Storage**: PostgreSQL levantado localmente con Docker Compose; `PostgreSQLMetricsStore`
cargará datos del Sprint y persistirá el resultado de métricas mediante `database/sql` y pgx

**Testing**: `go test ./...` con pruebas unitarias, escenarios BDD y pruebas de integración
contra PostgreSQL para `PostgreSQLMetricsStore`

**Target Platform**: Aplicación Go multiplataforma; el núcleo no depende del sistema operativo

**Project Type**: Núcleo de dominio para una aplicación de gestión de proyectos

**Performance Goals**: Calcular las métricas de un Sprint en menos de 2 segundos bajo uso normal

**Constraints**: Resultados reproducibles, aritmética decimal controlada, sin división por cero,
datos inválidos bloquean el cierre y los cálculos parciales deben mostrar `X de Y historias`

**Scale/Scope**: Un Sprint y sus historias terminadas; no incluye UI, reportes ni comparación
histórica entre Sprints

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Value-Driven Product Scope**: PASS. El plan implementa velocidad y desviación, directamente
  vinculadas al seguimiento y la planificación del equipo.
- **Quality by Construction**: PASS. El cálculo queda aislado en reglas de dominio pequeñas,
  explícitas y reutilizables.
- **Test-First and Evidence-Driven Delivery**: PASS. Cada regla tendrá pruebas unitarias,
  escenarios Given–When–Then y validación de persistencia PostgreSQL con `go test ./...`.
- **Agile, Collaborative Execution**: PASS. El alcance está limitado a una historia de usuario
  demostrable y revisable dentro de un Sprint.
- **Measurement and Continuous Improvement**: PASS. Las métricas y alertas definidas son el
  resultado principal de la feature.
- **Technology & Quality Constraints**: PASS. Se mantiene Go y se incorpora la implementación
  concreta PostgreSQL exigida por la constitución v1.1.0.

## Project Structure

### Documentation (this feature)

```text
specs/004-calculate-metrics/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── checklists/requirements.md
```

No se crea `contracts/` porque el repositorio no define todavía una API, CLI o interfaz externa.

### Source Code (repository root)

```text
go.mod
src/
└── metrics/
    ├── model.go
    ├── service.go
    ├── store.go
    ├── postgres_metrics_store.go
    ├── service_test.go
    └── postgres_metrics_store_integration_test.go
test/
└── metrics_bdd_test.go
```

**Structure Decision**: Se mantiene la estructura existente `src/` y `test/`, agregando un
paquete `src/metrics`. `service.go` contiene las reglas de cálculo y validación; `store.go`
define la interfaz; `postgres_metrics_store.go` implementa `PostgreSQLMetricsStore` con pgx
mediante `database/sql`; y su prueba verifica el adaptador contra PostgreSQL.

## Complexity Tracking

No hay violaciones de la constitución que requieran justificar complejidad adicional.

## Post-Design Constitution Check

- **Value-Driven Product Scope**: PASS. El diseño calcula las métricas necesarias para evaluar
  el cumplimiento del Sprint sin ampliar el alcance a reportes o comparaciones históricas.
- **Quality by Construction**: PASS. Cálculo, validación y persistencia quedan separados en
  servicio, modelo e interfaz/adaptador PostgreSQL.
- **Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas unitarias,
  escenarios BDD y una prueba de integración contra PostgreSQL.
- **Agile, Collaborative Execution**: PASS. Las métricas resultantes permiten inspeccionar la
  ejecución del Sprint y orientar la retrospectiva.
- **Measurement and Continuous Improvement**: PASS. La feature implementa directamente velocidad
  y desviación, incluyendo cálculo parcial y ausencia de estimación.
- **Technology & Quality Constraints**: PASS. Se usa Go y PostgreSQL obligatorio mediante pgx,
  `database/sql` y un adaptador concreto antes de integrar.
