# Feature Specification: Horas estimadas por historia (Estimaciones)

**Feature Branch**: `HU-17`

**Created**: 2026-10-21

**Status**: Draft

**Input**: User description: "HU-17 / Registro de Horas Estimadas por Historia: Como Product Builder, quiero registrar y modificar las horas estimadas de una historia desde la web, para poder compararlas con las horas reales. Depende de HU-02, HU-03, HU-22, HU-20a y HU-20b."

> **Alcance**: HU-17 permite registrar y modificar las horas estimadas de una historia después de su
> alta, en una pantalla propia, "Estimaciones", que lista las historias del proyecto con sus horas
> estimadas y reales. No incluye cargar las horas al crear la historia (HU-22), editar otros datos de
> la historia (HU-13), registrar horas reales (HU-03 y HU-24) ni calcular la desviación (HU-07).
>
> **Una sola fuente del dato**: la estimación se guarda en `user_stories.estimated_hours`, la columna
> que lee HU-07. Solo se modifica esa columna.
>
> **Regla de horas**: la validación (mayor que 0, hasta dos decimales, coma o punto, máximo 9999,99)
> es la que HU-22 deja en el dominio, junto a la regla de HU-02. HU-17 la reutiliza y no escribe otra.
>
> **Puntos que estaban "a definir"**: una historia de un Sprint finalizado se bloquea (propuesta del
> equipo que se confirma al refinar).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Registrar y modificar la estimación (Priority: P1)

Como Product Builder, quiero guardar y cambiar las horas estimadas de una historia, para poder
compararlas con las horas reales.

**Why this priority**: Es el flujo principal: HU-07 y HU-25 necesitan horas estimadas para calcular la
desviación de esfuerzo.

**Independent Test**: Con un proyecto con historias, guardar horas en una historia sin estimar,
modificarlas y verificar el mensaje, el valor mostrado y que las horas reales no cambian; repetir con
coma y con punto.

**Acceptance Scenarios**:

1. **Given** una historia sin horas estimadas, **When** ingreso "8" horas estimadas y presiono
   "Guardar", **Then** veo "Horas estimadas registradas" y la historia muestra 8 horas estimadas.
2. **Given** una historia con horas estimadas y horas reales registradas, **When** ingreso un valor
   nuevo válido y presiono "Guardar", **Then** veo "Horas estimadas actualizadas", el valor reemplaza
   al anterior y las horas reales no cambian.
3. **Given** una historia, **When** ingreso "1,5" o "1.5" horas estimadas y presiono "Guardar",
   **Then** se guarda 1,5 exacto, sin redondeo, y se muestra con coma.

---

### User Story 2 - Rechazos y errores (Priority: P1)

Como Product Builder, quiero que los valores inválidos y los cambios no permitidos se rechacen con un
motivo y sin tocar el valor anterior, para no falsear la comparación con las horas reales.

**Why this priority**: Una estimación inválida o cambiada después del cierre distorsiona la desviación
que calcula HU-07.

**Independent Test**: Enviar horas negativas, cero, vacías, no numéricas, con tres decimales y mayores
que 9999,99; enviar el cambio de una historia de un Sprint finalizado; simular una falla de guardado.

**Acceptance Scenarios**:

1. **Given** una historia con horas estimadas, **When** ingreso un valor negativo, cero, vacío, no
   numérico o con más de dos decimales, **Then** veo "El valor de horas no es válido" junto al campo,
   el valor anterior no cambia y lo que escribí se conserva.
2. **Given** una historia, **When** ingreso un valor mayor que 9999,99, **Then** veo "Las horas
   estimadas no pueden superar 9999,99" junto al campo y el valor anterior no cambia.
3. **Given** una historia de un Sprint finalizado, **When** se envía el cambio de sus horas estimadas,
   **Then** se bloquea con "La historia pertenece a un Sprint finalizado; sus horas estimadas no se
   pueden modificar" y el valor no cambia.
4. **Given** un valor válido, **When** falla el guardado, **Then** veo "La operación no pudo
   completarse", el valor anterior se conserva y no veo mensaje de éxito.

---

### User Story 3 - Ver las estimaciones (Priority: P1)

Como Product Builder, quiero ver todas las historias con sus horas estimadas y reales, para saber qué
falta estimar y qué no se puede cambiar.

**Why this priority**: Sin la pantalla no hay dónde registrar ni modificar las horas, y la lista es el
punto de partida de cada cambio.

**Independent Test**: Con historias estimadas, sin estimar y de un Sprint finalizado, verificar la
lista, "Sin estimar", el campo deshabilitado y el estado vacío.

**Acceptance Scenarios**:

1. **Given** un proyecto con historias, **When** abro "Estimaciones", **Then** veo cada historia con su
   título, estado, Story Points, horas estimadas y horas reales registradas.
2. **Given** una historia sin horas estimadas, **When** abro "Estimaciones", **Then** la veo como "Sin
   estimar" y no como cero.
3. **Given** una historia de un Sprint finalizado, **When** abro "Estimaciones", **Then** su campo de
   horas aparece deshabilitado con el motivo "Sprint finalizado".
4. **Given** un proyecto sin historias, **When** abro "Estimaciones", **Then** veo "Aún no hay
   historias".

---

### Edge Cases

- Un valor vacío se rechaza como "no numérico"; quitar una estimación ya cargada no está incluido.
- Guardar el mismo valor que ya tiene se acepta y muestra "Horas estimadas actualizadas".
- La coma y el punto se aceptan como separador decimal y se muestran con coma; no se aceptan
  separadores de miles y un valor con más de dos decimales se rechaza, no se redondea.
- Una historia sin horas reales muestra 0 en las reales; las estimadas ausentes se muestran "Sin
  estimar", nunca 0.
- Las historias del Sprint activo, de cualquier estado, y las del Backlog se pueden modificar; solo se
  bloquean las asignadas a un Sprint finalizado. Al cerrar el Sprint (HU-15), las no terminadas vuelven a
  "Pendiente" sin asignación y se pueden editar de nuevo.
- Si entre la lectura y el guardado el Sprint pasa a finalizado, el cambio se rechaza (la comprobación
  es atómica).
- Si dos personas guardan a la vez sobre la misma historia, gana la última; es aceptable porque el
  valor se reemplaza entero.
- Un identificador de historia no numérico, inexistente o de otro proyecto se rechaza con "La historia
  no existe"; el proyecto sale siempre del contexto del servidor.
- Un título con HTML o scripts se muestra como texto (regla de HU-20a y HU-20b).
- Cambiar la estimación no modifica las horas reales, los Story Points ni ningún otro campo.
- Recargar la página después de guardar no repite el cambio ni el mensaje (patrón de HU-20b; T040).
- Sin proyecto activo válido, HU-20b redirige a la selección de proyecto y no se guarda nada (T041).
- Si falla la lectura de las historias, se muestra la página de error de HU-20a (T032).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST ofrecer la sección "Estimaciones" en el menú, que requiere un proyecto
  activo.
- **FR-002**: La pantalla MUST listar las historias del proyecto activo con título, estado, Story
  Points, horas estimadas y horas reales registradas, con las más recientes primero.
- **FR-003**: Una historia sin horas estimadas MUST mostrarse "Sin estimar" y MUST NOT mostrarse como
  cero.
- **FR-004**: Guardar horas válidas en una historia sin estimación MUST registrarlas y mostrar "Horas
  estimadas registradas".
- **FR-005**: Guardar horas válidas en una historia con estimación MUST reemplazar la anterior, mostrar
  "Horas estimadas actualizadas" y MUST NOT modificar las horas reales ni otros campos.
- **FR-006**: El sistema MUST aceptar la coma y el punto como separador decimal, guardar el valor exacto
  sin redondeo y mostrarlo con coma.
- **FR-007**: Ante un valor negativo, cero, vacío, no numérico o con más de dos decimales, el sistema
  MUST mostrar "El valor de horas no es válido" junto al campo, conservar lo escrito y MUST NOT cambiar
  el valor anterior.
- **FR-008**: Ante un valor mayor que 9999,99, el sistema MUST mostrar "Las horas estimadas no pueden
  superar 9999,99" junto al campo y MUST NOT cambiar el valor anterior.
- **FR-009**: Una historia asignada a un Sprint finalizado MUST rechazar el cambio con "La historia
  pertenece a un Sprint finalizado; sus horas estimadas no se pueden modificar", sin cambios; la
  comprobación MUST ser atómica con el guardado.
- **FR-010**: La lista MUST mostrar el campo de una historia de un Sprint finalizado deshabilitado, con
  el motivo "Sprint finalizado".
- **FR-011**: Ante una falla de guardado, el sistema MUST mostrar "La operación no pudo completarse",
  conservar el valor anterior y MUST NOT mostrar éxito.
- **FR-012**: Sin historias, la pantalla MUST mostrar "Aún no hay historias".
- **FR-013**: El proyecto MUST salir del contexto del servidor y MUST NOT tomarse del formulario; una
  historia inexistente o de otro proyecto MUST rechazarse con "La historia no existe".
- **FR-014**: La validación de las horas MUST ser la regla única del dominio de HU-22 y el valor MUST
  guardarse en `user_stories.estimated_hours`, la misma fuente que lee HU-07.
- **FR-015**: Si falla la lectura de las historias, el sistema MUST mostrar la página de error de la
  aplicación y MUST NOT mostrar una lista incompleta.
- **FR-016**: Los mensajes de error MUST NOT exponer detalles internos de la base de datos.

### Especificación SDD *(requerida por el DoD)*

**Objetivo**: permitir registrar y modificar las horas estimadas de una historia, con una única regla
de validación y una única fuente del dato, para que la comparación con las horas reales de HU-07 sea
confiable.

**Entradas**:

| Formulario | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `POST /estimaciones` | `historia_id` | identificador numérico | sí | Del proyecto activo |
| `POST /estimaciones` | `horas_estimadas` | número positivo | sí | Hasta dos decimales, máximo 9999,99, coma o punto; vacío es inválido |

**Salidas esperadas**:

- Guardado válido: redirección 303 a `/estimaciones` y mensaje de una sola vista ("registradas" o
  "actualizadas").
- Valor inválido o historia inexistente: respuesta 422 con la lista, lo escrito conservado y el mensaje
  junto al campo de esa fila.
- Historia de un Sprint finalizado: respuesta 409 con la lista y el mensaje.
- Falla de guardado: respuesta 500 con la lista y "La operación no pudo completarse".
- Falla de lectura: página de error de HU-20a.
- Vista: historias con horas estimadas (o "Sin estimar"), horas reales y un campo por fila.

**Reglas de negocio**:

- Horas válidas: mayor que 0, hasta dos decimales, máximo 9999,99, coma o punto (regla de HU-22).
- Modificar reemplaza el valor anterior y solo toca `estimated_hours`.
- No se modifica una historia asignada a un Sprint finalizado.
- Las horas reales son las de HU-03 y se muestran solo para comparar.

**Restricciones**:

- Sin migraciones ni dependencias nuevas; no se modifican archivos de HU-02, HU-20a ni HU-20b: se
  agregan archivos nuevos.
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript y sin conexión a internet (HU-20a).

**Casos límite**: ver Edge Cases.

**Condiciones de error**:

| Condición | Campo | Mensaje | Respuesta |
|---|---|---|---|
| Valor negativo, cero, vacío, no numérico o con más de dos decimales | `horas_estimadas` | "El valor de horas no es válido" | 422 |
| Valor mayor que 9999,99 | `horas_estimadas` | "Las horas estimadas no pueden superar 9999,99" | 422 |
| Historia inexistente, no numérica o de otro proyecto | fila | "La historia no existe" | 422 |
| Historia de un Sprint finalizado | fila | "La historia pertenece a un Sprint finalizado; sus horas estimadas no se pueden modificar" | 409 |
| Falla de guardado | general | "La operación no pudo completarse" | 500 |
| Falla de lectura | página | Página de error de HU-20a | 500 o 503 |
| Sin proyecto activo válido | redirección | Mensaje de HU-20b | 303 |

Los textos propios de HU-17 son una propuesta del equipo y se definen en un solo lugar
(`estimates_form.go`); los de horas inválidas, máximo y "Sin estimar" son los de HU-22.

### Key Entities *(include if feature involves data)*

- **Estimación de horas**: Valor positivo con dos decimales guardado en `user_stories.estimated_hours`,
  opcional.
- **Fila de estimación**: Historia con título, estado, Story Points, horas estimadas, horas reales y
  si está bloqueada.
- **Horas reales**: Suma de los registros de HU-03 de la historia, solo lectura.
- **Proyecto activo**: Proyecto elegido en HU-20b sobre el que trabaja la pantalla.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de los valores entre 0,01 y 9999,99 con hasta dos decimales (coma o punto) se
  guarda exacto y se muestra con coma.
- **SC-002**: El 100% de los valores inválidos o por encima del máximo se rechaza con su mensaje
  conservando el valor anterior y lo escrito.
- **SC-003**: El 100% de los intentos de modificar una historia de un Sprint finalizado se rechaza sin
  cambios, desde la pantalla y con un envío directo.
- **SC-004**: Modificar la estimación nunca cambia las horas reales ni otro campo de la historia.
- **SC-005**: El 100% de las fallas de guardado muestra "La operación no pudo completarse", sin éxito y
  con el valor anterior intacto.
- **SC-006**: Ninguna historia sin estimar se muestra como 0.
- **SC-007**: La pantalla responde en menos de 500 ms bajo condiciones normales (la prueba de T028 lo
  mide con la base local y menos de 50 historias).

## Assumptions

- HU-22 está integrada en `main`: su regla de horas (rechazo de más de dos decimales, coma decimal y
  máximo de 9999,99) y el texto "Sin estimar" existen en el dominio; si no, T001 lo detecta.
- La columna `user_stories.estimated_hours` (`NUMERIC(6,2)`, mayor que 0) viene de la migración `002`
  de HU-02 y HU-07 la lee; no hace falta migración.
- Las horas reales son la suma de los registros de `effort_logs` de HU-03; no se modifican.
- Al cerrar un Sprint, HU-15 quita la asignación de las historias no terminadas; las "Hecho" quedan en el
  Sprint finalizado y por lo tanto bloqueadas.
- No hay roles ni identificación para este cambio; la autenticación y los roles se resuelven después del
  Sprint 2 (decisión del equipo).
- Mensajes, formularios y protección contra envíos ajenos vienen de HU-20b; la estructura común y los
  errores, de HU-20a; el proyecto activo, de HU-20b.

## Trazabilidad *(Historia → Spec → Criterios → Escenarios BDD → Tests → Código)*

| Escenario del Issue | Requisitos | Tests | BDD | Código |
|---|---|---|---|---|
| 1. Estimación registrada | FR-004, FR-014 | T005, T006, T012 | T015 | `src/story` (dominio), `estimates_handlers.go` |
| 2. Modificar la estimación | FR-005 | T005, T006, T013 | T015 | `src/story`, `estimates_handlers.go` |
| 3. Horas con coma decimal | FR-006 | T005, T014 | T015 | `src/story`, `estimates_form.go` |
| 4. Horas inválidas | FR-007 | T005, T020 | T025 | `src/story`, `estimates_form.go` |
| 5. Horas por encima del máximo | FR-008 | T005, T021 | T025 | `src/story`, `estimates_form.go` |
| 6. Historia de un Sprint finalizado | FR-009, FR-013 | T005, T006, T022, T023 | T025 | `postgres_estimated_hours_store.go`, `estimates_form.go` |
| 7. Error al guardar | FR-011, FR-016 | T024, T027 | T025 | `estimates_handlers.go` |
| 8. Lista de estimaciones | FR-002 | T008, T028 | T033 | `postgres_estimate_catalog.go`, `estimates.html` |
| 9. Historia sin estimar | FR-003 | T008, T029 | T033 | `estimates.html` |
| 10. Historia bloqueada en la lista | FR-010 | T008, T030 | T033 | `postgres_estimate_catalog.go`, `estimates.html` |
| 11. Sin historias | FR-012 | T031 | T033 | `estimates.html` |
| (HU-20b) Recarga sin repetir | FR-001 | T040 | | `estimates_handlers.go` |
| (HU-20b) Sin proyecto activo | FR-013 | T041 | | `estimates_handlers.go` |
| (HU-20a) Error al leer | FR-015 | T032 | | `estimates_handlers.go` |