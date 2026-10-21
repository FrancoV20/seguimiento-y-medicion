# Feature Specification: Interfaz web de proyectos

**Feature Branch**: `HU-21`

**Created**: 2026-10-18 | **Revised**: 2026-10-20

**Status**: Draft

**Input**: User description: "HU-21 / Interfaz Web de Proyectos (presenta HU-01): Como Usuario, quiero crear y ver proyectos desde una pantalla web, para hacerlo sin comandos. Depende de HU-01, FIX-HU-01, HU-20a y de la parte de formularios y mensajes de HU-20b."

> **Alcance**: HU-21 presenta en el navegador lo que HU-01 ya resuelve (modelo, validaciones, alta y
> persistencia). Muestra la lista de proyectos y el formulario de alta en una sola pantalla,
> "Gestión de Proyectos". No incluye modificar proyectos (HU-11), consultar su estado (HU-12),
> seleccionar el proyecto activo (HU-20b) ni dar de alta integrantes (ver Assumptions).
>
> **Regla de fechas**: la fecha de fin debe ser posterior a la de inicio. La regla es de HU-01 y
> cambió con **FIX-HU-01** (decisión del equipo del 20/10/2026): una fecha de fin igual a la de
> inicio ya no se acepta. HU-21 no se puede integrar antes de ese fix.
>
> **Pendientes de HU-01 que se cierran acá**: el acceso al formulario desde Gestión de Proyectos
> (FR-001 de HU-01) y los criterios de usabilidad SC-003 y SC-004 (tiempo para completar el alta y
> éxito al primer intento).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Crear un proyecto desde el formulario web (Priority: P1)

Como Usuario, quiero completar nombre, integrantes y fechas en un formulario y crear el proyecto,
para hacerlo sin comandos.

**Why this priority**: Es el flujo principal de HU-01 hecho consumible: sin esta pantalla el
sistema no tiene proyectos con los que trabajar y HU-22 a HU-25 no tienen datos que mostrar.

**Independent Test**: Con integrantes cargados en el sistema, completar el formulario con datos
válidos y verificar que el proyecto aparece en la lista con el mensaje de éxito; repetir con fechas
iguales (se rechaza), recargando la página y sin integrantes disponibles.

**Acceptance Scenarios**:

1. **Given** que estoy en "Gestión de Proyectos" y existen integrantes disponibles, **When**
   completo nombre, integrantes y fechas válidas y presiono "Crear", **Then** veo el mensaje
   "Proyecto creado correctamente" y el proyecto aparece en la lista.
2. **Given** que completo nombre e integrantes y ingreso la misma fecha de inicio y de fin, **When**
   presiono "Crear", **Then** veo junto a la fecha de fin el mensaje de fechas de HU-01, los datos
   se conservan y no se crea el proyecto.
3. **Given** que no hay integrantes cargados en el sistema, **When** abro "Gestión de Proyectos",
   **Then** veo "No hay integrantes disponibles" y no puedo crear un proyecto.

---

### User Story 2 - Corregir errores sin perder los datos (Priority: P1)

Como Usuario, quiero ver cada error junto a su campo y conservar lo que escribí, para corregirlo
sin volver a cargar todo.

**Why this priority**: Un formulario que borra los datos al fallar hace que el alta sea tediosa y
propensa a abandonarse; las reglas de HU-01 deben verse tal como están definidas.

**Independent Test**: Enviar el formulario con cada dato inválido por separado y todos juntos;
verificar los mensajes, que los datos se conservan y que no se crea ningún proyecto. Enviar una
fecha mal formada y un integrante inexistente. Simular una falla de guardado y verificar el mensaje
general.

**Acceptance Scenarios**:

1. **Given** que dejo el nombre vacío o con solo espacios y el resto de los datos son válidos,
   **When** presiono "Crear", **Then** veo el mensaje de HU-01 que indica que el nombre es
   obligatorio junto al nombre, los datos se conservan y no se crea el proyecto.
2. **Given** que ingreso una fecha de fin anterior a la de inicio y el resto de los datos son
   válidos, **When** presiono "Crear", **Then** veo el mensaje de fechas de HU-01 junto a la fecha de
   fin, los datos se conservan y no se crea el proyecto.
3. **Given** que dejo vacía la fecha de inicio o la de fin y el resto de los datos son válidos,
   **When** presiono "Crear", **Then** veo "Falta la fecha de inicio" o "Falta la fecha de fin",
   según cuál falte, junto a ese campo, los datos se conservan y no se crea el proyecto.
4. **Given** que hay integrantes disponibles y no marco ninguno, **When** presiono "Crear",
   **Then** veo el mensaje de HU-01 que indica que se requiere al menos un integrante junto al campo
   de integrantes, los datos se conservan y no se crea el proyecto.
5. **Given** un envío del formulario con una fecha que no tiene el formato AAAA-MM-DD, **When** se
   procesa, **Then** veo "La fecha no tiene un formato válido" junto a esa fecha, los datos se
   conservan y no se crea el proyecto.
6. **Given** un envío del formulario con un integrante que no existe o que no es numérico, **When**
   se procesa, **Then** veo "El integrante seleccionado no es válido" junto al campo de
   integrantes, los datos se conservan y no se crea el proyecto.
7. **Given** un formulario válido, **When** falla el guardado, **Then** veo "La operación no pudo
   completarse", los datos se conservan, no veo mensaje de éxito y el proyecto no aparece en la
   lista.

---

### User Story 3 - Ver la lista de proyectos (Priority: P2)

Como Usuario, quiero ver los proyectos existentes en la misma pantalla, para saber qué hay creado
antes de agregar uno nuevo.

**Why this priority**: Confirma visualmente el resultado del alta y orienta al equipo, pero no
bloquea la creación.

**Independent Test**: Con dos proyectos de distintas fechas y cantidad de integrantes, abrir la
pantalla y verificar columnas y orden; con la base sin proyectos, verificar el mensaje y el
formulario.

**Acceptance Scenarios**:

1. **Given** proyectos existentes, **When** abro "Gestión de Proyectos", **Then** veo cada proyecto
   con su nombre, fechas de inicio y fin y cantidad de integrantes, con el último proyecto creado
   primero.
2. **Given** que no hay proyectos, **When** abro la pantalla, **Then** veo el mensaje "Aún no hay
   proyectos" y el formulario para crear el primero.

---

### Edge Cases

- Un nombre con solo espacios se trata como vacío (regla de HU-01; se verifica en T001).
- Una fecha de fin igual a la de inicio se rechaza (FIX-HU-01), igual que en los Sprints; una fecha
  de fin un día posterior se acepta.
- Los nombres de proyectos pueden repetirse: HU-01 no define unicidad.
- Si el formulario envía el mismo integrante más de una vez, se toma una sola vez.
- Si el envío no incluye alguno de los campos, se trata como vacío y se informa junto al campo.
- Las fechas se ingresan con el control de fecha del navegador (`AAAA-MM-DD`); cualquier otro
  formato se rechaza como formato inválido aunque el navegador no lo haya impedido.
- Un nombre con HTML o scripts se muestra como texto en la lista (regla de HU-20a/HU-20b).
- Crear un proyecto no lo selecciona como proyecto activo (HU-20b: sin autoselección).
- Un nombre de más de 255 caracteres se rechaza con un mensaje junto al campo en lugar de fallar
  al guardar (extra, ver T027; no es criterio de aceptación).
- La lista no tiene paginación; con muchos proyectos se muestra completa.
- Recargar la página después de crear un proyecto no crea un segundo proyecto ni repite el mensaje
  (patrón de HU-20b, "Envío válido sin reenvío"; se prueba en T012).
- Si falla la lectura de proyectos o integrantes, se muestra la página de error de la aplicación y no
  un formulario incompleto (HU-20a; se prueba en T031).
- Cada escenario de error se prueba por separado (T019 a T022).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST ofrecer el acceso al formulario de alta desde la sección "Proyectos"
  del menú, cuya pantalla se titula "Gestión de Proyectos".
- **FR-002**: El formulario MUST pedir nombre, integrantes (selección múltiple) y fechas de inicio
  y de fin.
- **FR-003**: El selector de integrantes MUST ofrecer solo integrantes existentes en el sistema.
- **FR-004**: Cuando no existan integrantes, el sistema MUST mostrar "No hay integrantes
  disponibles" y MUST NOT permitir crear el proyecto.
- **FR-005**: Un alta válida MUST crear el proyecto según las reglas de HU-01, mostrar
  "Proyecto creado correctamente" una sola vez y mostrar el proyecto en la lista.
- **FR-006**: Ante datos inválidos, el sistema MUST mostrar los mensajes de HU-01 junto a cada
  campo, conservar los datos ingresados y MUST NOT crear el proyecto.
- **FR-007**: Cuando falte la fecha de inicio o la de fin, el sistema MUST indicar cuál falta
  ("Falta la fecha de inicio" o "Falta la fecha de fin").
- **FR-008**: El sistema MUST rechazar fechas con formato inválido con el mensaje "La fecha no
  tiene un formato válido" junto al campo.
- **FR-009**: El sistema MUST rechazar integrantes no numéricos o inexistentes con el mensaje "El
  integrante seleccionado no es válido" junto al campo de integrantes.
- **FR-010**: El sistema MUST rechazar una fecha de fin igual o anterior a la de inicio con el
  mensaje de fechas de HU-01, junto a la fecha de fin (regla de FIX-HU-01).
- **FR-011**: La lista MUST mostrar nombre, fechas de inicio y fin y cantidad de integrantes de
  cada proyecto, con el último proyecto creado primero.
- **FR-012**: Cuando no existan proyectos, el sistema MUST mostrar "Aún no hay proyectos" y el
  formulario de alta.
- **FR-013**: Ante una falla de guardado, el sistema MUST mostrar "La operación no pudo
  completarse", conservar los datos, MUST NOT mostrar éxito y MUST NOT mostrar el proyecto en la
  lista.
- **FR-014**: Recargar la página después de crear un proyecto MUST NOT crear un segundo proyecto.
- **FR-015**: Crear un proyecto MUST NOT cambiar el proyecto activo.
- **FR-016**: Si falla la lectura de proyectos o integrantes, el sistema MUST mostrar la página de
  error de la aplicación y MUST NOT mostrar un formulario incompleto.
- **FR-017**: Los mensajes de error MUST NOT exponer detalles internos de la base de datos.

### Especificación SDD *(requerida por el DoD)*

**Objetivo**: permitir crear proyectos y ver la lista de proyectos desde una pantalla web, con las
reglas de HU-01 y sin que la persona pierda lo que escribió ante un error.

**Entradas** (formulario `POST /proyectos`):

| Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|
| `nombre` | texto | sí | Sin espacios en los extremos; vacío o solo espacios se trata como vacío |
| `fecha_inicio` | fecha `AAAA-MM-DD` | sí | |
| `fecha_fin` | fecha `AAAA-MM-DD` | sí | Posterior a `fecha_inicio` |
| `integrantes` | identificadores numéricos (uno por integrante marcado) | sí, al menos uno | Los repetidos se toman una vez |

**Salidas esperadas**:

- Alta válida: redirección 303 a `/proyectos`, mensaje de éxito de una sola vista y el proyecto en
  la lista.
- Datos inválidos: respuesta 422 con el formulario, los datos conservados y un mensaje junto a cada
  campo con error.
- Falla de guardado: respuesta 500 con el formulario, los datos conservados y "La operación no pudo
  completarse".
- Falla de lectura: página de error de HU-20a.
- Lista: nombre, fechas `dd/mm/aaaa` y cantidad de integrantes; último creado primero.

**Reglas de negocio**:

- Las reglas de nombre, fechas e integrantes son las de HU-01 (con FIX-HU-01) y las aplica
  `ProjectService.CreateProject`; el handler solo valida el formato del envío.
- Un proyecto necesita al menos un integrante existente.
- Crear un proyecto no cambia el proyecto activo.

**Restricciones**:

- Sin migraciones ni dependencias nuevas; no se modifican HU-20a ni HU-20b.
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript y sin conexión a internet (HU-20a).

**Casos límite**: ver Edge Cases.

**Condiciones de error**:

| Condición | Campo | Mensaje | Respuesta |
|---|---|---|---|
| Nombre vacío o solo espacios | `nombre` | El de HU-01 | 422 |
| Fecha de fin anterior o igual a la de inicio | `fecha_fin` | El de fechas de HU-01 (FIX-HU-01) | 422 |
| Fecha de inicio vacía | `fecha_inicio` | "Falta la fecha de inicio" | 422 |
| Fecha de fin vacía | `fecha_fin` | "Falta la fecha de fin" | 422 |
| Formato de fecha inválido | la fecha afectada | "La fecha no tiene un formato válido" | 422 |
| Ningún integrante | `integrantes` | El de HU-01 | 422 |
| Integrante inexistente o no numérico | `integrantes` | "El integrante seleccionado no es válido" | 422 |
| Falla de guardado | general | "La operación no pudo completarse" | 500 |
| Falla de lectura | página | Página de error de HU-20a | 500 o 503 |

Los mensajes de HU-01 se toman del código de HU-01 (T001). Los textos propios de HU-21 y el mensaje
de éxito son una propuesta del equipo; se definen en un solo lugar (`projects_form.go`).

### Key Entities *(include if feature involves data)*

- **Proyecto**: Definido por HU-01; esta historia lo crea desde el formulario y lo lista.
- **Integrante disponible**: Integrante existente en el sistema que puede asignarse a un proyecto.
- **Fila de la lista**: Nombre, fechas y cantidad de integrantes de un proyecto.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de las altas válidas aparece en la lista con los datos ingresados.
- **SC-002**: El 100% de las entradas inválidas, incluidas las fechas de fin iguales a la de inicio,
  se rechaza con el mensaje junto a su campo, conservando los datos y sin crear proyectos.
- **SC-003**: Al menos el 90% de las personas de prueba completa el alta sin ayuda en el primer
  intento. *(Se verifica manualmente en la Sprint Review; confirmar el umbral contra la spec de
  HU-01.)*
- **SC-004**: El tiempo promedio para completar el alta es menor a 2 minutos. *(Se verifica
  manualmente en la Sprint Review; confirmar el umbral contra la spec de HU-01.)*
- **SC-005**: El 100% de las fallas de guardado muestra "La operación no pudo completarse", sin
  éxito y sin proyecto en la lista.
- **SC-006**: La pantalla responde en menos de 500 ms bajo condiciones normales de uso (la prueba
  de T029 la mide con la base local y menos de 50 proyectos).
- **SC-007**: El 100% de las recargas posteriores a un alta no crea proyectos duplicados.

## Assumptions

- El modelo, las validaciones, el caso de uso de alta y la persistencia de proyectos son los de
  HU-01 con el cambio de FIX-HU-01; esta historia no define otras reglas ni mensajes de HU-01.
- FIX-HU-01 está integrada en `main` antes de integrar HU-21; si no lo estuviera, T011 falla.
- Los integrantes ya existen en el sistema: la tabla de integrantes se alimenta fuera de HU-01. En
  el Sprint 2 se cargan con un script semilla (`db/seeds/members.sql`), que no es una migración.
  El alta de integrantes desde la aplicación no tiene historia todavía (pendiente del equipo).
- El estado de un proyecto (No iniciado, En curso, Finalizado) es de HU-12; esta lista no lo
  muestra.
- Mensajes, formularios y protección contra envíos ajenos vienen de HU-20b (US1 y US2); la
  estructura común y los errores, de HU-20a.
- Seleccionar el proyecto activo se hace desde la pantalla de HU-20b; esta lista no incluye el
  botón de selección ni reutiliza esa pantalla.
- Modificar proyectos, quitar integrantes, paginación y filtros quedan fuera del alcance.
- No hay roles: cualquier persona que use la aplicación puede crear proyectos.

## Trazabilidad *(Historia → Spec → Criterios → Escenarios BDD → Tests → Código)*

| Escenario del Issue | Requisitos | Tests de handler y unitarios | BDD | Código |
|---|---|---|---|---|
| 1. Alta exitosa | FR-001 a FR-003, FR-005, FR-015 | T009, T010 | T014 | `projects_form.go`, `projects_handlers.go`, `projects.html` |
| 2. Fecha de fin igual a la de inicio | FR-010 | T011 | T014 | `projects_form.go` (mapeo del error de HU-01) |
| 3. Sin integrantes disponibles | FR-004 | T013 | T014 | `projects_handlers.go`, `projects.html` |
| 4. Nombre vacío | FR-006 | T019 | T024 | `projects_form.go` |
| 5. Fecha de fin anterior a la de inicio | FR-006, FR-010 | T019 | T024 | `projects_form.go` |
| 6. Fecha de inicio o de fin faltante | FR-007 | T020 | T024 | `projects_form.go` |
| 7. Sin integrantes marcados | FR-006 | T019 | T024 | `projects_form.go` |
| 8. Fecha con formato inválido | FR-008 | T009, T021 | T024 | `projects_form.go` |
| 9. Integrante inválido | FR-009 | T022 | T024 | `projects_form.go` |
| 10. Error al guardar el proyecto | FR-013, FR-017 | T023, T028 | T024 | `projects_handlers.go` |
| 11. Lista de proyectos | FR-011 | T006, T029 | T032 | `postgres_project_catalog.go`, `projects.html` |
| 12. Lista vacía | FR-012 | T030 | T032 | `projects.html` |
| (HU-20b) Recarga sin duplicar | FR-014 | T012 | | `projects_handlers.go` |
| (HU-20a) Error al leer | FR-016 | T031 | | `projects_handlers.go` |
