# Implementation Plan: Registro y trazabilidad de defectos

**Branch**: `HU-06` | **Date**: 2026-10-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar el registro de defectos en un paquete de dominio nuevo (`src/defect`) y presentarlo en una pantalla
propia, "Defectos" (`/defectos`), como páginas generadas por el servidor. El servicio `Register` valida la
descripción, la severidad y la historia devolviendo todas las violaciones juntas y delega en un puerto de
almacenamiento (`Store`) con su adaptador PostgreSQL. El adaptador, en una sola transacción, bloquea el Sprint
activo del proyecto con `FOR SHARE` (así no se mezcla con el cierre de HU-15), comprueba que la historia sea del
proyecto e inserta el defecto "Abierto" con el Sprint de detección. Una migración nueva crea la tabla `defects`.
La pantalla tiene una sección con `RequiresProject: true`, un catálogo propio de solo lectura (lista de defectos,
opciones de historia y número del Sprint activo) y handlers que solo arman el envío, llaman al servicio y
traducen sus errores. No modifica HU-02, HU-04, HU-15, HU-20a ni HU-20b.

> **Verificación inicial (T001)**: en `main`, los paquetes son `src/backlog` y `src/sprint`; las tablas son
> `user_stories` y `sprints`. `projects.id` es `SERIAL`/`INTEGER`; `user_stories.id` y `sprints.id` son
> `BIGSERIAL`/`BIGINT`, aunque los modelos Go usan `int64`. Los Sprints admiten `'Activo'` y `'Finalizado'`;
> las historias admiten `'Pendiente'`, `'En Sprint'`, `'En progreso'` y `'Hecho'`. No hay paquete ni
> identificadores de defectos existentes en `src`; los nombres nuevos con prefijo `Defect` no colisionan.
> No se encontró una operación de cierre de Sprint en `main`; los bloqueos `FOR UPDATE` existentes son de
> asignación/remoción de historias, por lo que el contrato de HU-15 sigue pendiente.
>
> **Setup verificado (T002-T004)**: los contratos comunes están disponibles. Los parciales son definiciones
> `alert` y `field` dentro de `templates/layout.html` (no archivos separados `alert.html` y `field.html`).
> `origin/main` llega a `011` (HU-05a); `012` es candidato, sin reserva en ramas remotas visibles. HU-15 todavía no tiene
> responsable ni implementación, por lo que su cierre con `FOR UPDATE` no pudo verificarse en el código. Ver
> [data-model.md](data-model.md); coordinar nuevamente el número y el contrato cuando HU-15 se asigne.

## Technical Context

**Language/Version**: Go 1.27.1

**Primary Dependencies**: Biblioteca estándar de Go (`net/http`, `html/template`, `database/sql`, `unicode/utf8`) y
`github.com/jackc/pgx/v5/stdlib` ya incorporado. No se agregan dependencias nuevas.

**Storage**: PostgreSQL con Docker Compose. Una migración nueva crea `defects`. El registro usa una transacción de
`database/sql`; el catálogo solo lee `defects`, `user_stories` y `sprints`.

**Testing**: `go test -count=1 -p 1 ./...` con pruebas del servicio y del adaptador contra PostgreSQL real, pruebas
de handler con `net/http/httptest` y `net/http/cookiejar`, y escenarios BDD en `test/web_defects_bdd_test.go`.

**Target Platform**: Aplicación Go multiplataforma; desarrollo en Windows con Docker Desktop.

**Project Type**: Regla de dominio, migración y pantalla de una aplicación web SSR.

**Performance Goals**: La pantalla responde en menos de 500 ms bajo uso normal (SC-006).

**Constraints**: Todo o nada al registrar, un solo Sprint activo (FIX-HU-04), sin mezclarse con el cierre de
Sprint, ningún detalle interno en las respuestas, y proyecto y Sprint de detección tomados siempre del servidor.

**Scale/Scope**: Una pantalla, dos rutas y una tabla. No incluye resolver ni reabrir (HU-19), editar ni borrar, ni
métricas de defectos (HU-10).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Value-Driven Product Scope**: PASS. Habilita HU-19, HU-10, HU-08 y HU-09 y cubre el requisito mínimo de
  gestión de defectos del enunciado.
- **II. Quality by Construction**: PASS. Las reglas y el servicio están en el dominio, el almacenamiento detrás de
  un puerto y la pantalla solo traduce.
- **III. Test-First and Evidence-Driven Delivery**: PASS. Cada escenario tiene pruebas contra PostgreSQL real y un
  escenario BDD; las pruebas se escriben antes de implementar.
- **IV. Agile, Collaborative Execution**: PASS. Alcance demostrable en la Sprint Review; la lista (P2) se puede
  dejar fuera sin afectar el registro.
- **V. Measurement and Continuous Improvement**: PASS. Genera los datos de defectos detectados y resueltos de las
  métricas.
- **VI. PostgreSQL Persistence**: PASS. `Store` tiene adaptador PostgreSQL con pruebas de integración y la migración
  tiene `up` y `down` probados.
- **Technology & Quality Constraints**: PASS. Go y PostgreSQL se mantienen; sin dependencias nuevas.

## Project Structure

### Documentation (this feature)

```text
specs/006-defects/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
└── checklists/requirements.md
```

No se crea `contracts/`: la interfaz externa son páginas HTML; su comportamiento está en `spec.md` y las
direcciones y campos, abajo.

### Source Code (repository root)

```text
db/migrations/
├── NNN_create_defects.up.sql             # NNN = último de main + 1
└── NNN_create_defects.down.sql
src/
├── defect/                               # paquete de dominio nuevo
│   ├── model.go                          # Severity, Status, RegisterInput, Registered
│   ├── errors.go                         # errores del dominio
│   ├── store.go                          # puerto Store
│   ├── service.go                        # Register y validación completa
│   ├── service_test.go
│   ├── postgres_store.go                 # adaptador PostgreSQL (transacción)
│   └── *_integration_test.go
└── web/
    ├── defects_catalog.go                # tipos y la interfaz DefectCatalog
    ├── postgres_defects_catalog.go       # adaptador PostgreSQL (solo lectura)
    ├── defects_form.go                   # campos, textos, armado del envío y errores por campo
    ├── defects_handlers.go               # GET /defectos y POST /defectos
    ├── templates/defects.html            # formulario y lista
    └── *_test.go
test/
└── web_defects_bdd_test.go
```

**Structure Decision**: Paquete de dominio nuevo `src/defect`, archivos nuevos en `src/web` con el prefijo `Defect`
y una migración. Se edita `src/cmd/server/main.go` para construir el servicio y el catálogo y registrar la
sección. No se editan archivos de otras historias.

### Direcciones y campos (propuestos)

| Método | Dirección | Efecto |
| --- | --- | --- |
| GET | `/defectos` | Formulario (si hay Sprint activo) y lista de defectos |
| POST | `/defectos` | Campos `descripcion`, `severidad` e `historia`; registra el defecto; 303 a `/defectos` |

La sección se registra como `Section{Name: "Defectos", Path: "/defectos", RequiresProject: true}`. No requiere
integrante. El proyecto sale del servidor.

### Flujo del POST

1. Si no hay proyecto activo válido, HU-20b redirige con 303 a la selección de proyecto.
2. Armar el envío: `historia` numérica (vacía o no numérica genera el error de ese campo), `descripcion` sin
   espacios en los bordes y `severidad` tal como llegó.
3. Pedir al servicio la validación completa de descripción, severidad e historia y unir los errores (uno por campo).
4. Con errores: pantalla con los datos conservados y 422.
5. Sin errores: llamar al servicio con el proyecto del contexto. Éxito: `SetFlash` con el mensaje y el número del
   Sprint y redirección 303 a `/defectos`.
6. `ErrNoActiveSprint`: pantalla con el mensaje y 409. `ErrStoryNotFound`: error junto a `historia` y 422.
   Cualquier otro error: `MsgOperationFailed` con la pantalla y 500.

### Mapeo de errores del dominio a mensajes

| Error del dominio | Campo | Mensaje |
| --- | --- | --- |
| `ErrDescriptionRequired` | `descripcion` | "La descripción es obligatoria" |
| `ErrDescriptionTooLong` | `descripcion` | "La descripción no puede superar 1000 caracteres" |
| `ErrSeverityRequired` | `severidad` | "Seleccione la severidad" |
| `ErrSeverityInvalid` | `severidad` | "La severidad no es válida" |
| `ErrStoryRequired` | `historia` | "Seleccione la historia relacionada" |
| `ErrStoryNotFound` | `historia` | "La historia no pertenece al proyecto" |
| `ErrNoActiveSprint` | (general) | "No hay un Sprint activo; no se puede registrar un defecto" |
| Cualquier otro | (general) | `MsgOperationFailed` |

### Contratos que consume

- De HU-20a: `Server.Register(section, routes)`, `Renderer.Page`, `Fail` y `RenderError`.
- De HU-20b: `SetFlash`, `MsgOperationFailed`, `FormView`, la redirección 303, los parciales `alert.html` y
  `field.html`, la protección contra envíos de otro origen y `ActiveContextFrom`.
- De HU-04 y FIX-HU-04: el esquema de Sprints y la regla de un solo Sprint activo.
- De HU-15: el cierre del Sprint bloquea la fila del Sprint (`FOR UPDATE`); el registro la toma `FOR SHARE`.

## Complexity Tracking

| Decisión | Por qué hace falta | Alternativa descartada |
|---|---|---|
| Migración con la tabla `defects` | Los defectos no existen en el esquema y deben guardarse con su historia y sus Sprints | Guardarlos dentro de `user_stories` (descartada: un defecto no es una historia y HU-10 los cuenta por Sprint) |

## Post-Design Constitution Check

- **I. Value-Driven Product Scope**: PASS. El diseño se limita a registrar y listar; resolver es de HU-19.
- **II. Quality by Construction**: PASS. Servicio, almacenamiento, catálogo, formulario y handlers separados.
- **III. Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas del servicio, del
  adaptador, del catálogo, de handler y BDD.
- **IV. Agile, Collaborative Execution**: PASS. Las tres historias de usuario se entregan de forma incremental.
- **V. Measurement and Continuous Improvement**: PASS. Sin cambios en métricas; las habilita.
- **VI. PostgreSQL Persistence**: PASS. Puerto y catálogo con pruebas contra PostgreSQL y migración probada.
- **Technology & Quality Constraints**: PASS. Sin dependencias ni tecnologías nuevas.