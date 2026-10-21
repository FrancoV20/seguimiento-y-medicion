# Feature Specification: FIX-HU-04 / Un solo Sprint activo por proyecto y estado "En Sprint"

**Feature Branch**: `FIX-HU-04`

**Created**: 2026-10-20

**Status**: Draft

**Input**: Decisión del equipo del 20/10/2026. Este fix corrige una historia ya integrada; las specs originales de la historia se actualizan en la tarea T008.

## Por qué se hace

- HU-04 permite varios Sprints activos en un proyecto (dos de sus escenarios lo suponen). HU-12, HU-14, HU-23 y HU-24 hablan de "el Sprint activo" en singular. Si la regla vive solo en la pantalla (HU-23), cualquier otro camino puede crear dos Sprints activos y se rompe la separación entre lógica de negocio y capa web que pide el DoD.
- Según la spec de HU-23, HU-04 ya deja las historias asignadas en estado "En Sprint" (migración `004`), pero el texto de HU-04 dice solo "actualiza su estado". Este fix deja el texto y los tests alineados con los cuatro estados.

## Acceptance Scenarios

Los escenarios que no se nombran acá no cambian: "Fechas de sprint inválidas", "Sprint sin historias asignadas", "Identificador automático del Sprint" y "Sprint Goal obligatorio".

**Escenario nuevo: Un solo Sprint activo por proyecto.**
Dado un proyecto que tiene el Sprint activo "Sprint N",
Cuando el Scrum Master intenta crear otro Sprint,
Entonces el sistema impide la creación, informa que ya hay un Sprint activo (Sprint N) y no crea el Sprint ni asigna historias.

**Escenario nuevo: Un rechazo no deja huecos en la numeración.**
Dado que se rechazó un intento de crear un Sprint porque ya había uno activo,
Cuando el Sprint activo pasa a finalizado y creo un Sprint nuevo,
Entonces el nuevo Sprint tiene el número siguiente al del último Sprint creado, sin saltos.

**Escenario nuevo: Sprint nuevo cuando todos los anteriores están finalizados.**
Dado un proyecto cuyos Sprints están todos finalizados,
Cuando creo un Sprint con datos válidos,
Entonces el Sprint se crea como Sprint activo.

**Escenario nuevo: La regla es por proyecto.**
Dado que el proyecto A tiene un Sprint activo y el proyecto B no tiene ninguno,
Cuando creo un Sprint en el proyecto B,
Entonces el Sprint se crea.

**Escenario nuevo: Creación simultánea.**
Dado un proyecto sin Sprint activo,
Cuando llegan al mismo tiempo dos solicitudes para crear un Sprint,
Entonces se crea un solo Sprint y la otra solicitud se rechaza porque ya hay un Sprint activo.

**Escenario modificado: Asignación exitosa.**
Dado un proyecto con historias en el Product Backlog,
Cuando creo un Sprint con su "Sprint Goal" y le asigno 3 historias pendientes,
Entonces el sistema asocia esas historias al Sprint y las deja en estado "En Sprint".

**Escenario reescrito: Intento de asignar una historia que ya está asignada a un Sprint.**
(Reemplaza a "Intento de asignar una historia que ya está en otro Sprint activo".)
Dado que una historia ya está asignada a un Sprint,
Cuando el Scrum Master intenta asignarla a un Sprint,
Entonces el sistema bloquea la acción y muestra un mensaje indicando que la historia ya está asignada al Sprint N.

**Escenario reescrito: Quitar una historia y volver a asignarla.**
(Reemplaza a "Mover una historia entre Sprints", que ya no aplica con un solo Sprint activo.)
Dado una historia en estado "En Sprint" del Sprint activo,
Cuando el Scrum Master la quita del Sprint y luego la asigna de nuevo al Sprint activo,
Entonces al quitarla la historia vuelve a "Pendiente" y la nueva asignación se permite sin conflicto.

## Especificación SDD *(requerida por el DoD)*

**Objetivo**: garantizar en el dominio que un proyecto tiene como máximo un Sprint activo, y que las historias asignadas quedan en estado "En Sprint".

**Entradas**: proyecto, Sprint Goal, fechas del Sprint y las historias a asignar (puede no haber ninguna).

**Salidas esperadas**:
- Sin Sprint activo en el proyecto: se crea el Sprint con el número siguiente y las historias quedan "En Sprint".
- Con un Sprint activo: error `ErrActiveSprintExists` con el número del Sprint activo; no se crea el Sprint, no se asigna ninguna historia y no se consume un número.

**Reglas de negocio**:
- R1: un proyecto tiene como máximo un Sprint con estado `Activo`; los Sprints finalizados no cuentan.
- R2: la verificación y la toma del número ocurren en la misma transacción, con la fila del proyecto bloqueada.
- R3: un rechazo no incrementa `projects.next_sprint_number`.
- R4: una historia se puede asignar solo si está "Pendiente" y sin asignación; al asignarla pasa a "En Sprint"; al quitarla vuelve a "Pendiente".
- R5: la regla es por proyecto: un Sprint activo en el proyecto A no afecta al proyecto B.

**Restricciones**: sin migraciones; no se cambia la validación de `RemoveStory`; no se agregan dependencias; la capa web no decide esta regla.

**Casos límite**: dos creaciones simultáneas en el mismo proyecto; proyecto con todos los Sprints finalizados; historia "Hecho" de un Sprint finalizado (sigue asignada a él); mismo número de Sprint en proyectos distintos.

**Condiciones de error**: Sprint activo existente; historia ya asignada ("ya está asignada al Sprint N"); falla de guardado: no queda Sprint ni asignación parcial, como hoy.

## Alcance

**Entra:**
1. La regla "un solo Sprint activo por proyecto" en el dominio de HU-04, con su error.
2. Los tests de HU-04 que creaban dos Sprints activos en el mismo proyecto.
3. Los textos de HU-04 con los cuatro estados.

**No entra:**
- HU-23: usa este error para mostrar "Ya hay un Sprint activo (Sprint N)". Su spec se corrige en su propia historia.
- Validar qué historias se pueden quitar (`RemoveStory`): depende de una decisión pendiente del equipo (historia "Quitar"). No se toca.
- Cerrar un Sprint (HU-15) y los cambios de texto de HU-14, HU-08, HU-13 y HU-15 por los cuatro estados: se hacen en cada una de esas historias.
- Sin migraciones: la regla se aplica en el servicio con bloqueo del proyecto (ver T005). No se agrega un índice único en la base.

## Cómo debe resolverse la regla (propuesta)

Crear el Sprint ya ocurre dentro de una transacción que incrementa el contador de Sprints del proyecto (`projects.next_sprint_number`). En esa misma transacción: bloquear la fila del proyecto (`SELECT … FOR UPDATE`), verificar que no existe un Sprint del proyecto con estado `Activo` y recién entonces tomar el número. Si existe, devolver un error propio (por ejemplo `ErrActiveSprintExists`) que lleve el número del Sprint activo, sin haber incrementado el contador. El bloqueo hace que dos solicitudes simultáneas se ordenen: la segunda ve el Sprint que creó la primera. Los nombres exactos se confirman en T001.

## Success Criteria

- **SC-001**: El 100% de los intentos de crear un segundo Sprint activo en un proyecto se rechaza con el número del Sprint activo.
- **SC-002**: Dos creaciones simultáneas en el mismo proyecto producen un solo Sprint y numeración sin huecos.
- **SC-003**: Toda historia asignada queda en "En Sprint" y vuelve a "Pendiente" al quitarla.
- **SC-004**: Ningún test del repositorio depende de dos Sprints activos en el mismo proyecto y `go test ./...` pasa completo.

## Trazabilidad *(Escenario → Tareas)*

| Escenario | Tareas |
|---|---|
| Un solo Sprint activo por proyecto | T002, T005 |
| Un rechazo no deja huecos en la numeración | T002, T005 |
| Sprint nuevo cuando todos los anteriores están finalizados | T002 |
| La regla es por proyecto | T002 |
| Creación simultánea | T003, T005 |
| Asignación exitosa (modificado) | T004 |
| Intento de asignar una historia que ya está asignada a un Sprint | T004, T006 |
| Quitar una historia y volver a asignarla | T004 |
