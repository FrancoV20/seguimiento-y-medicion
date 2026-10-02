# Implementation Plan: Agregar historia al Product Backlog

**Branch**: `002-add-backlog-story` | **Date**: 2026-10-01 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar el caso de uso de alta de una historia en el Product Backlog de un proyecto
activo. El núcleo validará título, descripción, prioridad y criterios de aceptación, aceptará
Story Points ausentes con una alerta y restringirá los valores informados a la escala Fibonacci
definida. `PostgreSQLBacklogStore` persistirá la historia y su asociación con PostgreSQL usando
pgx mediante `database/sql`. La operación creará siempre la historia en estado `Pendiente` y
será atómica.

## Technical Context

**Language/Version**: Go 1.22 o posterior, fijado en `go.mod` durante la inicialización

**Primary Dependencies**: Biblioteca estándar de Go, `github.com/jackc/pgx/v5` y
`github.com/jackc/pgx/v5/stdlib` para registrar pgx mediante `database/sql`

**Storage**: PostgreSQL levantado localmente con Docker Compose; `PostgreSQLBacklogStore`
implementará el almacenamiento mediante `database/sql` y pgx, con transacciones para la
historia y su asociación al proyecto

**Testing**: `go test ./...` con pruebas unitarias, escenarios BDD y pruebas de integración
contra PostgreSQL para `PostgreSQLBacklogStore`

**Target Platform**: Aplicación Go multiplataforma; el dominio no depende del sistema operativo

**Project Type**: Núcleo de dominio para una aplicación de gestión de proyectos

**Performance Goals**: Validar y agregar una historia en menos de 100 ms bajo uso normal

**Constraints**: Campos obligatorios validados, criterios no vacíos, estado inicial fijo,
Story Points opcional, valores informados limitados a Fibonacci y sin duplicación en reintentos

**Scale/Scope**: Un proyecto activo y su Product Backlog; no incluye UI, edición, eliminación,
ordenamiento ni transición posterior de estados

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Value-Driven Product Scope**: PASS. La feature agrega una unidad de trabajo necesaria para
  planificar el Product Backlog.
- **Quality by Construction**: PASS. Las validaciones y transiciones se concentran en reglas de
  dominio explícitas, sin duplicar responsabilidades.
- **Test-First and Evidence-Driven Delivery**: PASS. Se cubrirán los escenarios de creación,
  criterios obligatorios, Story Points opcionales, valores Fibonacci y persistencia PostgreSQL
  con `go test ./...`.
- **Agile, Collaborative Execution**: PASS. La historia produce una unidad de backlog lista
  para priorización y planificación posterior.
- **Measurement and Continuous Improvement**: PASS. Story Points y prioridad quedan disponibles
  para las métricas y decisiones de planificación futuras.
- **Technology & Quality Constraints**: PASS. Se mantiene Go, SDD/BDD/TDD y se incorpora la
  implementación concreta PostgreSQL exigida por la constitución v1.1.0.

## Project Structure

### Documentation (this feature)

```text
specs/002-add-backlog-story/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── checklists/requirements.md
```

No se crea `contracts/` porque el repositorio aún no define una API, CLI o interfaz externa.

### Source Code (repository root)

```text
go.mod
src/
└── backlog/
    ├── model.go
    ├── service.go
    ├── store.go
    ├── postgres_backlog_store.go
    ├── service_test.go
    └── postgres_backlog_store_integration_test.go
test/
└── backlog_bdd_test.go
```

**Structure Decision**: Se conserva la estructura existente `src/` y `test/`, agregando el
paquete `src/backlog`. `service.go` contiene el caso de uso, `store.go` define la interfaz,
`postgres_backlog_store.go` implementa `PostgreSQLBacklogStore` con pgx mediante `database/sql`
y su prueba verifica el adaptador contra PostgreSQL. Las pruebas unitarias viven junto al
paquete; `test/` queda reservado para escenarios de aceptación de mayor nivel.

## Complexity Tracking

No hay violaciones de la constitución que requieran justificar complejidad adicional.

## Post-Design Constitution Check

- **Value-Driven Product Scope**: PASS. El diseño se limita al alta de historias y sus reglas
  de validación, sin incluir edición, eliminación ni ordenamiento avanzado.
- **Quality by Construction**: PASS. El modelo separa proyecto, backlog, solicitud, historia y
  resultado, y mantiene el estado inicial bajo control del dominio.
- **Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define casos verificables
  para creación, criterios obligatorios, ausencia de Story Points y escala Fibonacci.
- **Agile, Collaborative Execution**: PASS. El resultado es una historia pendiente lista para
  priorización y planificación del equipo.
- **Measurement and Continuous Improvement**: PASS. Story Points y prioridad quedan disponibles
  para futuras métricas sin introducir cálculos en esta feature.
- **Technology & Quality Constraints**: PASS. El diseño usa Go y PostgreSQL como persistencia
  obligatoria, con `PostgreSQLBacklogStore`, pgx mediante `database/sql` y pruebas de integración
  antes de integrar la feature.
