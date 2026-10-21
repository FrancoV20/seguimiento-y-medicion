# Feature Specification: FIX-HU-01 / Fecha de fin posterior a la de inicio en proyectos

**Feature Branch**: `FIX-HU-01`

**Created**: 2026-10-20

**Status**: Draft

**Input**: Decisión del equipo del 20/10/2026. Este fix corrige una historia ya integrada; las specs originales de la historia se actualizan en la tarea T008.

## Por qué se hace

HU-01 hoy permite crear un proyecto con la misma fecha de inicio y de fin (escenario "Fecha de finalización igual a la fecha de inicio") y la migración `001` tiene `CHECK (end_date >= start_date)`. HU-04 rechaza el fin "anterior o igual" al inicio en los Sprints. Con la regla actual puede existir un proyecto en el que no cabe ningún Sprint. El equipo decidió que el fin debe ser posterior al inicio.

## Acceptance Scenarios

**Escenario: Fechas inválidas (se mantiene).**
Dado que el usuario intenta crear un proyecto,
Cuando ingresa una fecha de finalización anterior a la fecha de inicio,
Entonces el sistema impide la creación y muestra un mensaje de error indicando la inconsistencia temporal.

**Escenario: Fecha de finalización igual a la fecha de inicio (reemplaza al escenario que la permitía).**
Dado que el usuario intenta crear un proyecto con nombre e integrantes válidos,
Cuando ingresa la misma fecha para inicio y finalización,
Entonces el sistema impide la creación y muestra un mensaje indicando que la fecha de finalización debe ser posterior a la fecha de inicio.

**Escenario: Fecha de finalización un día posterior a la de inicio (límite nuevo).**
Dado que el usuario intenta crear un proyecto con nombre e integrantes válidos,
Cuando ingresa una fecha de finalización un día posterior a la de inicio,
Entonces el sistema registra el proyecto y muestra un mensaje de éxito.

**Escenario: La base también rechaza fechas iguales.**
Dado un intento de guardar un proyecto con la misma fecha de inicio y de finalización directamente en el almacenamiento,
Cuando se ejecuta el guardado,
Entonces el almacenamiento lo rechaza y no queda ningún proyecto nuevo.

## Especificación SDD *(requerida por el DoD)*

**Objetivo**: que el alta de un proyecto exija una fecha de finalización estrictamente posterior a la de inicio, tanto en el servicio como en la base de datos.

**Entradas**: nombre (texto), integrantes (identificadores), fecha de inicio y fecha de finalización (fechas de calendario, sin hora).

**Salidas esperadas**:
- Fechas válidas: el proyecto se registra y se informa el éxito, como hoy.
- Fecha de finalización anterior o igual a la de inicio: error de fechas ligado al campo de fecha de finalización; no se crea nada.

**Reglas de negocio**:
- R1: la fecha de finalización debe ser posterior a la de inicio (comparación estricta por fecha de calendario).
- R2: hay una sola función de validación de fechas; HU-11 (modificar proyecto) debe reutilizarla.
- R3: la base rechaza `end_date <= start_date`, como segunda barrera.

**Restricciones**: no se edita la migración `001`; no cambian los mensajes de nombre, integrantes ni fechas faltantes; no se agregan dependencias.

**Casos límite**: fin igual a inicio (se rechaza); fin un día posterior (se acepta); fin en otro mes o año (se acepta); fecha faltante (sigue rigiendo el escenario "Fecha de inicio o fin faltante").

**Condiciones de error**: fin menor o igual al inicio produce el error de fechas; una falla de guardado produce "La operación no pudo completarse" sin éxito, como hoy.

## Alcance

**Entra:**
1. La regla de dominio de HU-01: fin posterior al inicio.
2. Una migración nueva que cambia la restricción de la base.
3. Los tests de HU-01 y los de cualquier otra historia que creen proyectos con fechas iguales.
4. La documentación de HU-01 (spec, quickstart, tasks) y el texto de la historia.

**No entra** (se corrige en su propia historia):
- HU-21: su spec y su issue ya tienen la regla nueva.
- HU-11 (modificar proyecto): su escenario "Fechas inválidas al modificar" pasa a "anterior o igual".
- HU-12 no cambia: un proyecto que empieza o termina hoy sigue siendo "En curso".

## Texto del mensaje (propuesta)

"La fecha de finalización debe ser posterior a la fecha de inicio". Se usa tanto para fin anterior como para fin igual, así hay un solo mensaje para la regla. En T001 se confirma el texto actual y se reemplaza.

## Success Criteria

- **SC-001**: El 100% de los intentos de crear un proyecto con fecha de fin igual o anterior a la de inicio se rechaza, en el servicio y en la base.
- **SC-002**: El 100% de las pruebas de otras historias que creaban proyectos con fechas iguales fue corregido y `go test ./...` pasa completo.
- **SC-003**: La migración se aplica y se revierte sin errores (`migrate up`, `migrate down 1`, `migrate up`).

## Trazabilidad *(Escenario → Tareas)*

| Escenario | Tareas |
|---|---|
| Fechas inválidas (se mantiene) | T002, T004, T005 |
| Fecha de finalización igual a la fecha de inicio | T002, T004, T005 |
| Fecha de finalización un día posterior a la de inicio | T002, T004 |
| La base también rechaza fechas iguales | T003, T006 |
