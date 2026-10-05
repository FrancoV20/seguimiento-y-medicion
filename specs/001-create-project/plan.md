# Implementation Plan: Crear nuevo proyecto

**Branch**: `001-create-project` | **Date**: 2026-10-01 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Implementar el caso de uso de creación de proyectos desde Gestión de Proyectos. El dominio
validará nombre, integrantes y período, permitirá fechas iguales, rechazará una fecha final
anterior y persistirá el proyecto mediante `ProjectStore`, cuya implementación concreta usará
PostgreSQL con `github.com/jackc/pgx/v5` a través de `database/sql`. La operación será atómica:
ningún proyecto se registra cuando falla una validación o la persistencia.

## Technical Context

**Language/Version**: Go 1.27.1, fijado en `go.mod` durante la inicialización

**Primary Dependencies**: Biblioteca estándar de Go, `github.com/jackc/pgx/v5` y su adaptador
`github.com/jackc/pgx/v5/stdlib` para registrar pgx mediante `database/sql`

**Storage**: PostgreSQL levantado localmente con Docker Compose; `ProjectStore` tendrá un
adaptador concreto que use `database/sql` con el driver pgx y pruebas de integración contra esa
instancia

**Testing**: `go test ./...` con pruebas unitarias del dominio, escenarios BDD y pruebas de
integración PostgreSQL para el adaptador de `ProjectStore`

**Target Platform**: Aplicación Go multiplataforma; reglas de validación independientes del SO

**Project Type**: Núcleo de dominio con caso de uso de aplicación para gestión de proyectos

**Performance Goals**: Validar y registrar un proyecto en menos de 100 ms sin contar latencia
de la base de datos

**Constraints**: Nombre no vacío, al menos un integrante, fechas obligatorias, `end >= start`,
mensaje de éxito solo después de persistencia confirmada y sin duplicación ante errores

**Scale/Scope**: Un proyecto y sus integrantes; no incluye edición, eliminación, consulta,
autenticación ni migraciones fuera del esquema mínimo necesario para persistir el proyecto

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Value-Driven Product Scope**: PASS. La feature crea el contexto necesario para backlog,
  Sprints, esfuerzo y métricas posteriores.
- **Quality by Construction**: PASS. Validación, entidad y persistencia se separan mediante un
  caso de uso y un contrato de almacenamiento.
- **Test-First and Evidence-Driven Delivery**: PASS. Se probarán creación, campos obligatorios,
  fechas iguales, fechas inválidas y errores de persistencia con `go test ./...`, incluyendo
  integración del adaptador contra PostgreSQL.
- **Agile, Collaborative Execution**: PASS. El proyecto creado queda disponible para organizar
  el trabajo del equipo.
- **Measurement and Continuous Improvement**: PASS. El período e integrantes persistidos son
  la base de las métricas y el seguimiento posterior.
- **Technology & Quality Constraints**: PASS. Se mantiene Go, SDD/BDD/TDD y se cumple la
  persistencia obligatoria en PostgreSQL con una implementación concreta de `ProjectStore`.

## Project Structure

### Documentation (this feature)

```text
specs/001-create-project/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── checklists/requirements.md
```

No se crea `contracts/` porque `ProjectStore` es un contrato interno y el repositorio aún no
expone una API, CLI o interfaz externa.

```text
docker-compose.yml       # PostgreSQL para desarrollo e integración
go.mod
src/
└── project/
    ├── model.go
    ├── service.go
    ├── store.go
    ├── postgres_store.go
    ├── service_test.go
    └── postgres_store_integration_test.go
test/
└── project_bdd_test.go
```

**Structure Decision**: Se conserva la estructura existente `src/` y `test/`, agregando el
paquete `src/project`. `service.go` orquesta validación y persistencia; `store.go` define la
frontera; `postgres_store.go` implementa `ProjectStore` con pgx mediante `database/sql`; y la
prueba de integración verifica el adaptador contra PostgreSQL.

## Complexity Tracking

No hay violaciones de la constitución que requieran justificar complejidad adicional.

## Post-Design Constitution Check

- **Value-Driven Product Scope**: PASS. El diseño se limita al alta de proyectos y su período,
  integrantes y persistencia confirmada.
- **Quality by Construction**: PASS. La entidad, la solicitud, el servicio y `ProjectStore`
  tienen responsabilidades separadas y verificables.
- **Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas para éxito,
  fechas iguales, fechas inválidas, obligatorios y fallas de persistencia.
- **Agile, Collaborative Execution**: PASS. El proyecto creado queda listo para backlog y
  Sprints posteriores.
- **Measurement and Continuous Improvement**: PASS. El período y los integrantes persistidos
  permiten construir seguimiento y métricas posteriores.
- **Technology & Quality Constraints**: PASS. El diseño mantiene Go y usa PostgreSQL como
  persistencia obligatoria, con un adaptador concreto y pruebas de integración antes de integrar.
