# Implementation Plan: Registrar horas trabajadas

**Branch**: `003-timesheet` | **Date**: 2026-10-09 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar un servicio de dominio Go que registre el esfuerzo diario (integrante, fecha,
actividad y horas) sobre historias asignadas a un Sprint activo y acumule las horas reales por
historia. El servicio validará todos los campos antes de guardar, devolverá todos los errores
juntos, rechazará horas fuera del rango `(0, 24]` y nunca informará éxito ante una falla de
almacenamiento. `PostgreSQLEffortStore` persistirá los registros y calculará el acumulado
dentro de una transacción mediante pgx a través de `database/sql`; el acumulado será la fuente
de `actualHours` para HU-07.

## Technical Context

**Language/Version**: Go 1.27.1, fijado en `go.mod` durante la inicialización

**Primary Dependencies**: Biblioteca estándar de Go, `github.com/jackc/pgx/v5` y
`github.com/jackc/pgx/v5/stdlib` para registrar pgx mediante `database/sql`

**Storage**: PostgreSQL levantado localmente con Docker Compose; `PostgreSQLEffortStore`
guardará los registros de esfuerzo y leerá el contexto de historias, Sprints e integrantes
mediante `database/sql` y pgx

**Testing**: `go test ./...` con pruebas unitarias, escenarios BDD y pruebas de integración
contra PostgreSQL para `PostgreSQLEffortStore`

**Target Platform**: Aplicación Go multiplataforma; el núcleo no depende del sistema operativo

**Project Type**: Núcleo de dominio para una aplicación de gestión de proyectos

**Performance Goals**: Registrar un esfuerzo y devolver el acumulado en menos de 2 segundos bajo
uso normal

**Constraints**: Aritmética decimal exacta sin punto flotante, horas en `(0, 24]` con hasta dos
decimales, validación completa antes de guardar, ningún registro parcial ante fallas y mensajes
de error que no expongan detalles internos

**Scale/Scope**: Registros de esfuerzo de las historias de un Sprint activo; no incluye UI,
edición o eliminación de registros, reportes ni cálculo de métricas (HU-07)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Value-Driven Product Scope**: PASS. El plan implementa el registro de horas reales, base
  directa de la comparación entre esfuerzo estimado y real.
- **Quality by Construction**: PASS. Validación, aritmética de horas y persistencia quedan
  aisladas en piezas pequeñas, explícitas y reutilizables.
- **Test-First and Evidence-Driven Delivery**: PASS. Cada regla tendrá pruebas unitarias,
  escenarios Given–When–Then y validación de persistencia PostgreSQL con `go test ./...`.
- **Agile, Collaborative Execution**: PASS. El alcance es una historia de usuario demostrable
  y revisable dentro del Sprint 1.
- **Measurement and Continuous Improvement**: PASS. Las horas registradas alimentan las métricas
  de velocidad y desviación del Sprint.
- **Technology & Quality Constraints**: PASS. Se mantiene Go y se incorpora la implementación
  concreta PostgreSQL exigida por la constitución v1.1.0.

## Project Structure

### Documentation (this feature)

```text
specs/003-timesheet/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
└── checklists/requirements.md
```

No se crea `contracts/` porque el repositorio no define todavía una API, CLI o interfaz externa.

### Source Code (repository root)

```text
go.mod
db/
└── migrations/
    ├── 005_create_effort_logs.up.sql
    └── 005_create_effort_logs.down.sql
src/
└── effort/
    ├── model.go
    ├── errors.go
    ├── hours.go
    ├── validation.go
    ├── store.go
    ├── postgres_effort_store.go
    ├── service.go
    ├── test_store.go
    ├── hours_test.go
    ├── service_test.go
    └── postgres_effort_store_integration_test.go
test/
└── effort_bdd_test.go
```

**Structure Decision**: Se mantiene la estructura existente `src/` y `test/`, agregando un
paquete `src/effort`. `hours.go` contiene la aritmética de horas; `validation.go` las reglas de
campos; `service.go` el caso de uso; `store.go` define la interfaz; `postgres_effort_store.go`
implementa `PostgreSQLEffortStore` con pgx mediante `database/sql`; y su prueba verifica el
adaptador contra PostgreSQL.

## Complexity Tracking

No hay violaciones de la constitución que requieran justificar complejidad adicional.

## Post-Design Constitution Check

- **Value-Driven Product Scope**: PASS. El diseño registra y acumula horas reales sin ampliar el
  alcance a edición, reportes o cálculo de desviación.
- **Quality by Construction**: PASS. Validación, aritmética, servicio y persistencia quedan
  separados en archivos con responsabilidades únicas.
- **Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas unitarias,
  escenarios BDD y una prueba de integración contra PostgreSQL.
- **Agile, Collaborative Execution**: PASS. El acumulado resultante puede ser consumido por HU-07
  de forma independiente.
- **Measurement and Continuous Improvement**: PASS. La feature provee el dato `actualHours` que
  habilita las métricas del Sprint.
- **Technology & Quality Constraints**: PASS. Se usa Go y PostgreSQL obligatorio mediante pgx,
  `database/sql`, una restricción `CHECK` y un adaptador concreto antes de integrar.
