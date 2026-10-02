# Implementation Plan: Crear y asignar Sprint

**Branch**: `003-create-sprint` | **Date**: 2026-10-01 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar el caso de uso de creación de Sprints y asignación de historias pendientes del
Product Backlog. El servicio generará identificadores secuenciales no editables, exigirá un
Sprint Goal y un período válido, permitirá Sprints sin historias con advertencia y aplicará la
regla dura de una sola asignación por historia en Sprints activos. `PostgreSQLSprintStore`
persistirá el Sprint y sus asociaciones con PostgreSQL usando pgx mediante `database/sql`.
Todas las validaciones se resolverán antes de mutar el proyecto para evitar asociaciones parciales.

## Technical Context

**Language/Version**: Go 1.22 o posterior, fijado en `go.mod` durante la inicialización

**Primary Dependencies**: Biblioteca estándar de Go, `github.com/jackc/pgx/v5` y
`github.com/jackc/pgx/v5/stdlib` para registrar pgx mediante `database/sql`, más tipos de backlog

**Storage**: PostgreSQL levantado localmente con Docker Compose; `PostgreSQLSprintStore` usará
`database/sql` y pgx con transacciones para conservar identificador, estado, fechas y relaciones
exclusivas de historias

**Testing**: `go test ./...` con pruebas unitarias, escenarios BDD y pruebas de integración
contra PostgreSQL para `PostgreSQLSprintStore`

**Target Platform**: Aplicación Go multiplataforma; reglas de dominio independientes del SO

**Project Type**: Núcleo de dominio para una aplicación de gestión de proyectos

**Performance Goals**: Crear un Sprint y validar sus asignaciones en menos de 100 ms para un
backlog de hasta 1.000 historias

**Constraints**: Sprint Goal obligatorio, fecha final estrictamente posterior, identificador
secuencial automático, cero o más historias, exclusividad en Sprints activos y mutación atómica

**Scale/Scope**: Un proyecto y sus Sprints e historias; no incluye cierre, edición del Goal,
eliminación de Sprints ni interfaz externa

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Value-Driven Product Scope**: PASS. La feature enmarca trabajo del Product Backlog dentro
  de una iteración planificable.
- **Quality by Construction**: PASS. Las fechas, el identificador y la exclusividad quedan en
  reglas de dominio explícitas y centralizadas.
- **Test-First and Evidence-Driven Delivery**: PASS. Se probarán creación, fechas, Sprint Goal,
  advertencia sin historias, conflicto, movimiento y persistencia PostgreSQL con `go test ./...`.
- **Agile, Collaborative Execution**: PASS. El Sprint y sus historias representan una unidad
  de trabajo inspeccionable para el equipo.
- **Measurement and Continuous Improvement**: PASS. El estado `En Sprint` deja las historias
  preparadas para seguimiento y métricas posteriores.
- **Technology & Quality Constraints**: PASS. Se mantiene Go, SDD/BDD/TDD y se incorpora la
  implementación concreta PostgreSQL exigida por la constitución v1.1.0.

## Project Structure

### Documentation (this feature)

```text
specs/003-create-sprint/
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
└── sprint/
    ├── model.go
    ├── service.go
    ├── store.go
    ├── postgres_sprint_store.go
    ├── service_test.go
    └── postgres_sprint_store_integration_test.go
test/
└── sprint_bdd_test.go
```

**Structure Decision**: Se conserva la estructura existente `src/` y `test/`, agregando el
paquete `src/sprint`. `service.go` valida y orquesta; `store.go` define la interfaz;
`postgres_sprint_store.go` implementa `PostgreSQLSprintStore` con pgx mediante `database/sql`;
y su prueba verifica el adaptador contra PostgreSQL.

## Complexity Tracking

No hay violaciones de la constitución que requieran justificar complejidad adicional.

## Post-Design Constitution Check

- **Value-Driven Product Scope**: PASS. El diseño se limita a crear Sprints, asignar historias
  y permitir su movimiento explícito.
- **Quality by Construction**: PASS. El modelo separa proyecto, Sprint, historia, solicitud y
  resultado, con invariantes de fechas, estados y exclusividad.
- **Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas para Goal,
  fechas, creación vacía, conflictos, movimiento y ausencia de mutaciones parciales.
- **Agile, Collaborative Execution**: PASS. El Sprint enmarca trabajo del backlog en una unidad
  visible y planificable.
- **Measurement and Continuous Improvement**: PASS. El estado `En Sprint` deja las historias
  preparadas para seguimiento y métricas posteriores.
- **Technology & Quality Constraints**: PASS. El diseño conserva Go y usa PostgreSQL como
  persistencia obligatoria, con `PostgreSQLSprintStore`, pgx mediante `database/sql` y pruebas
  de integración antes de integrar la feature.
