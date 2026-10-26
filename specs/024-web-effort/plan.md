# Implementation Plan: Interfaz web de registro de esfuerzo

**Branch**: `HU-24` | **Date**: 2026-10-22 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar la pantalla "Registro de esfuerzo" (`/esfuerzo`) como páginas generadas por el servidor: una sola
ruta que muestra las historias del Sprint activo con sus horas acumuladas, el formulario de carga y el historial
de la historia elegida. La sección exige proyecto activo e integrante identificado, así que HU-20b redirige a
elegirlos cuando faltan. Los handlers no duplican reglas de negocio: arman el envío (fecha `AAAA-MM-DD`, horas con
coma o punto), comprueban que la historia sea del Sprint activo del proyecto, piden a HU-03 todas las violaciones
y las muestran junto a cada campo, y delegan el registro en el servicio de HU-03 con el integrante y el proyecto
del contexto del servidor. Un adaptador `PostgreSQLEffortCatalog` propio de HU-24 lee las historias del Sprint
activo, sus horas acumuladas y el historial. No agrega migraciones y no modifica HU-04, HU-14, HU-15, HU-20a ni
HU-20b; en HU-03 solo agrega una función de validación completa si no existiera.

> **Supuestos a verificar antes de implementar (T001, T002 y T004)**:
> 1. La ruta del paquete de esfuerzo (`src/effort` o similar) y del servicio de HU-03: entrada de registro,
>    tipo de las horas, errores y mensajes exactos (horas inválidas, actividad vacía, fecha requerida, integrante
>    no especificado), si ya existe una validación que devuelve todas las violaciones, qué historias acepta
>    (¿solo del Sprint activo del proyecto?, ¿incluye "Hecho"?) y si el servicio verifica el proyecto.
> 2. El esquema de `effort_logs` (migración `003`): columnas de historia, integrante, fecha, actividad y horas,
>    tipo de las horas y columna de orden de carga.
> 3. HU-20a y HU-20b completas en `main`: `Server.Register`, `Section.RequiresProject` y
>    `Section.RequiresMember`, `ActiveContextFrom` con identificador y nombre del integrante, `SetFlash`,
>    `FormView`, parciales `alert.html` y `field.html`, redirección 303 y regreso a la pantalla tras identificarse.
> 4. Los identificadores ya definidos en el paquete `web` por HU-14, HU-15, HU-17, HU-21, HU-22 y HU-23, para que
>    los nombres con prefijo `Effort` no choquen, y cómo formatean horas HU-17 y HU-22.

## Technical Context

**Language/Version**: Go 1.27.1

**Primary Dependencies**: Biblioteca estándar de Go (`net/http`, `html/template`, `time`) y
`github.com/jackc/pgx/v5/stdlib` ya incorporado. No se agregan dependencias nuevas.

**Storage**: PostgreSQL con Docker Compose. El registro usa el servicio y el store de HU-03.
`PostgreSQLEffortCatalog` solo lee `sprints`, `sprint_story_assignments`, `user_stories`, `effort_logs` y
`members` mediante `database/sql` y pgx.

**Testing**: `go test -count=1 -p 1 ./...` con pruebas unitarias del formulario, pruebas de handler con
`net/http/httptest` y `net/http/cookiejar` sobre el servicio de HU-03 y la base reales (el proyecto y el integrante
se eligen por el flujo de HU-20b), escenarios BDD en `test/web_effort_bdd_test.go` y pruebas de integración contra
PostgreSQL.

**Target Platform**: Aplicación Go multiplataforma; desarrollo en Windows con Docker Desktop.

**Project Type**: Pantalla de una aplicación web SSR sobre el núcleo de dominio en Go.

**Performance Goals**: La pantalla responde en menos de 500 ms bajo uso normal (SC-006).

**Constraints**: Sin migraciones ni dependencias nuevas, sin JavaScript, reglas y mensajes de HU-03 sin copiarse,
ningún detalle interno en las respuestas, los datos del formulario se conservan ante cualquier error, y el proyecto
y el integrante salen siempre del contexto del servidor.

**Scale/Scope**: Una pantalla y dos rutas. No incluye editar ni borrar registros, métricas (HU-07, HU-25) ni cambio
de estado de historias (HU-14).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Value-Driven Product Scope**: PASS. Hace consumible el registro de esfuerzo de HU-03 y alimenta HU-07 y
  HU-25; no agrega funcionalidades fuera de la historia.
- **II. Quality by Construction**: PASS. El armado del formulario, el adaptador de HU-03, el catálogo y los
  handlers están separados; las reglas siguen en HU-03.
- **III. Test-First and Evidence-Driven Delivery**: PASS. Cada escenario tiene pruebas de handler y un escenario BDD
  con almacenamiento real; las pruebas se escriben antes de implementar.
- **IV. Agile, Collaborative Execution**: PASS. Alcance demostrable en la Sprint Review; el historial (P2) se puede
  dejar fuera sin afectar la carga.
- **V. Measurement and Continuous Improvement**: PASS. Facilita cargar el esfuerzo real sobre el que se calcula la
  desviación.
- **VI. PostgreSQL Persistence**: PASS. La interfaz `EffortCatalog` tiene su adaptador `PostgreSQLEffortCatalog` con
  prueba de integración contra PostgreSQL; la escritura reutiliza el almacenamiento PostgreSQL de HU-03.
- **Technology & Quality Constraints**: PASS. Go y PostgreSQL se mantienen; sin dependencias nuevas.

## Project Structure

### Documentation (this feature)

```text
specs/024-web-effort/
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
src/
├── effort/                                   # solo si T010 lo requiere: validación completa de HU-03
└── web/
    ├── effort_form.go                        # campos, textos, armado del envío y errores por campo
    ├── effort_actions.go                     # interfaz EffortActions y adaptador sobre el servicio de HU-03
    ├── effort_catalog.go                     # tipos y la interfaz EffortCatalog
    ├── postgres_effort_catalog.go            # adaptador PostgreSQL (solo lectura)
    ├── effort_handlers.go                    # GET /esfuerzo y POST /esfuerzo
    ├── templates/effort.html                 # lista, formulario e historial
    └── *_test.go
test/
└── web_effort_bdd_test.go
```

**Structure Decision**: Todo lo nuevo vive en `src/web` con el prefijo `Effort` y no edita archivos de otras
historias. Se edita `src/cmd/server/main.go` para construir el catálogo y el adaptador y registrar la sección. Si
HU-03 no tiene una validación que devuelva todas las violaciones, se agrega esa función en su paquete (T010).

### Direcciones y campos (propuestos)

| Método | Dirección | Efecto |
| --- | --- | --- |
| GET | `/esfuerzo` | Lista de historias del Sprint activo con horas acumuladas, formulario e historial (si `historia` es válida) |
| GET | `/esfuerzo?historia=ID` | Igual, con esa historia elegida y su historial |
| POST | `/esfuerzo` | Campos `historia`, `fecha`, `actividad` y `horas`; registra la carga y redirige con 303 a `/esfuerzo?historia=ID` |

La sección se registra como `Section{Name: "Registro de esfuerzo", Path: "/esfuerzo", RequiresProject: true,
RequiresMember: true}`. El proyecto y el integrante se obtienen del servidor.

### Flujo del POST

1. Si no hay proyecto activo válido o integrante identificado, HU-20b redirige con 303.
2. Armar el envío: `historia` numérica, `fecha` en `AAAA-MM-DD` (vacía pasa a HU-03), `actividad` sin espacios en los
   bordes y `horas` con coma o punto; un valor que no se puede leer genera el error de ese campo.
3. Leer las historias del Sprint activo del proyecto; sin Sprint activo, 409 con el mensaje. Si `historia` falta o no
   está entre ellas, error junto a ese campo.
4. Pedir a HU-03 todas las violaciones de los campos que se pudieron leer y unirlas con las del paso 2 (un error
   por campo).
5. Con errores: pantalla con los datos conservados y 422.
6. Sin errores: llamar al servicio de HU-03 con proyecto e integrante del contexto. Éxito: `SetFlash` con el mensaje y
   redirección 303 a `/esfuerzo?historia=ID`.
7. Error de HU-03 que sea una violación o una historia fuera del Sprint (por ejemplo, el Sprint se cerró): 422 junto al
   campo que corresponda. Cualquier otro error: `MsgOperationFailed` con la pantalla y 500.

### Mapeo de errores de HU-03 a campos (verificado en T001)

| Error de HU-03 | Campo | Mensaje |
| --- | --- | --- |
| `ErrInvalidHours` (negativas, cero, más de 24, más de dos decimales, no numéricas) | `horas` | "El valor de horas no es válido" |
| `ErrActivityRequired` | `actividad` | "La descripción de la actividad es obligatoria" |
| `ErrDateRequired` | `fecha` | "La fecha es requerida" |
| `ErrDateInFuture` | `fecha` | "La fecha no puede ser posterior a la fecha actual" |
| `ErrMemberRequired` | (general) | "Se debe especificar el integrante responsable" |
| `ErrStoryNotInSprint` (el Sprint se cerró en el medio) | `historia` | "La historia no está asignada a un Sprint activo" |
| Historia fuera del Sprint activo del proyecto (control de la web) | `historia` | "La historia no pertenece al Sprint activo" |
| `ErrStorage` o cualquier otro | (general) | `MsgOperationFailed` |

> HU-03 sí valida que la fecha no sea futura (`ErrDateInFuture`); la pantalla muestra ese mensaje junto a `fecha`.

### Contratos que consume

- De HU-20a: `Server.Register(section, routes)`, `Renderer.Page`, `Fail` y `RenderError`.
- De HU-20b: `SetFlash`, `MsgOperationFailed`, `FormView`, la redirección 303, los parciales `alert.html` y
  `field.html`, la protección contra envíos de otro origen, `ActiveContextFrom` (proyecto e integrante) y la
  exigencia de proyecto e integrante por sección.
- De HU-03: el servicio de registro de esfuerzo, sus errores y mensajes y la validación de todas las violaciones.
- De HU-04 y FIX-HU-04: el esquema y la regla de un solo Sprint activo.

## Complexity Tracking

> Sin violaciones de la constitución que justificar. La única posible modificación a otra historia es agregar en
> HU-03 una función que devuelva todas las violaciones, si no existiera.

## Post-Design Constitution Check

- **I. Value-Driven Product Scope**: PASS. El diseño se limita a lo que dice la historia.
- **II. Quality by Construction**: PASS. Formulario, adaptador, catálogo y handlers separados; reglas en HU-03.
- **III. Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas unitarias, de handler,
  de integración y BDD.
- **IV. Agile, Collaborative Execution**: PASS. Las tres historias de usuario se entregan de forma incremental.
- **V. Measurement and Continuous Improvement**: PASS. Sin cambios en métricas; facilita el dato que usan.
- **VI. PostgreSQL Persistence**: PASS. Catálogo con adaptador y prueba de integración contra PostgreSQL.
- **Technology & Quality Constraints**: PASS. Sin dependencias ni tecnologías nuevas.