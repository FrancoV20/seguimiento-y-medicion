# Especificación: HU-13 – Consulta y Edición del Product Backlog (web)

**Rama**: `HU-13`
**Creada**: 2026-10-27
**Estado**: Borrador
**Constitución**: v1.1.0
**Depende de**: HU-02, HU-04, HU-20a, HU-20b, HU-22
**Orden de implementación**: después de HU-15 (cierre de Sprint) y HU-17 (horas estimadas), que tienen que estar en
`main` antes de crear la rama (decisión del equipo, 27/10; ver T001)
**Prioridad**: Alta (Must Have) · **Valor**: 85 · **Estimación**: 5 SP (sugerida)

**Entrada**: "HU-13 / Consulta y Edición del Product Backlog (web). Como Scrum Master, quiero ver el Product Backlog
de un proyecto ordenado por prioridad, filtrarlo por estado y editar los datos de una historia, para mantenerlo
ordenado y actualizado."

## Historia de usuario

Como Scrum Master, quiero ver el Product Backlog de un proyecto ordenado por prioridad, filtrarlo por estado y
editar los datos de una historia, para mantenerlo ordenado y actualizado.

> **Alcance**: HU-13 cambia el orden de la lista del Backlog de HU-22 (por prioridad y antigüedad), le agrega un
> filtro por estado y un enlace "Editar" en cada historia. El formulario de edición cambia el título, la
> descripción, la prioridad, los Story Points y los criterios de aceptación, con las reglas y los mensajes de HU-02
> y HU-22. Una historia que pertenece a un Sprint finalizado no se edita.
>
> **Contexto**: la sección "Backlog" exige proyecto activo (HU-20b); la lista y la edición trabajan solo sobre las
> historias de ese proyecto. No exige integrante identificado.

## Clarifications

### Session 2026-10-27

- Q: ¿Qué tiene que pasar cuando alguien abre la dirección de edición de una historia que pertenece a un Sprint
  finalizado? → A: Redirección 303 a `/backlog` con la advertencia "La historia pertenece a un Sprint finalizado;
  no se puede editar", una sola vez; no se muestra el formulario.
- Q: Si alguien filtra el Backlog, presiona "Editar" y guarda, ¿adónde vuelve? → A: A la lista con el mismo filtro.
  El enlace "Editar" lleva el estado, el formulario lo conserva en un campo oculto y, al guardar, cancelar o ser
  redirigido, se vuelve a `/backlog?estado=…`. Un estado inválido o ausente vuelve a la lista completa.

## Sección SDD

### Intención

Mantener el Product Backlog ordenado y al día: ver primero lo más prioritario, encontrar las historias por estado
y corregir sus datos sin perder lo que ya se planificó, ejecutó o midió (estado, horas estimadas, Sprints
finalizados).

### Alcance

- Lista del Backlog ordenada por prioridad (Alta, Media, Baja) y, a igual prioridad, la más antigua primero.
- Filtro por estado: "Todos", "Pendiente", "En Sprint", "En progreso" y "Hecho".
- Enlace "Editar" en cada historia; deshabilitado, con el motivo "Sprint finalizado", si la historia pertenece a un
  Sprint finalizado.
- Formulario de edición con el título, la descripción, la prioridad, los Story Points y los criterios actuales;
  "Agregar criterio" y "Quitar" sin guardar.
- Guardar los cinco datos juntos: todo o nada.
- Rechazos: título, descripción o criterios vacíos; prioridad o Story Points inválidos; historia de un Sprint
  finalizado; historia inexistente o de otro proyecto; falla al guardar.

### Fuera de alcance

- Editar las horas estimadas (HU-17), el estado (HU-14) o el Sprint de la historia (HU-23).
- Borrar historias.
- Ordenar a mano o por otro criterio; paginar.
- Filtrar por más de un estado a la vez o por otros datos (prioridad, texto).
- Avisar a otras personas que tienen el formulario abierto (última edición gana).

### Entradas

| Formulario | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `GET /backlog` | `estado` (en la dirección) | texto | no | "Pendiente", "En Sprint", "En progreso", "Hecho" o "Todos"; ausente = "Todos" |
| dirección | `{id}` | identificador numérico | sí | Historia a editar |
| `GET /backlog/{id}/editar` | `estado` (en la dirección) | texto | no | Filtro de la lista desde la que se abrió (R12) |
| `POST /backlog/{id}/editar` | `estado` (campo oculto) | texto | no | El mismo filtro; decide adónde se vuelve (R12) |
| `POST /backlog/{id}/editar` | `titulo` | texto | sí | Se recortan los espacios |
| `POST /backlog/{id}/editar` | `descripcion` | texto | sí | Se recortan los espacios |
| `POST /backlog/{id}/editar` | `prioridad` | "Alta", "Media" o "Baja" | sí | |
| `POST /backlog/{id}/editar` | `story_points` | número de la escala o vacío | no | Vacío = "Sin estimar" |
| `POST /backlog/{id}/editar` | `criterio` | texto (varios) | al menos uno no vacío | Los vacíos se descartan |
| `POST /backlog/{id}/editar` | `accion` | `guardar`, `agregar_criterio` o `quitar_criterio:N` | no | Ausente = `guardar` |

Los nombres de los campos y de las acciones son los de HU-22. El proyecto sale del contexto del servidor (HU-20b),
nunca del formulario.

### Salidas esperadas

- Lista: 200 con las historias del proyecto activo en el orden de R1, filtradas según `estado`.
- Estado inválido en la dirección: 200 con "El estado no es válido" y la lista completa.
- Filtro sin resultados: 200 con "No hay historias con ese estado".
- Proyecto sin historias: "Aún no hay historias" y el formulario de alta de HU-22.
- Formulario de edición: 200 con los datos actuales de la historia.
- "Agregar criterio" o "Quitar": 200 con el formulario y el cambio aplicado; no se guarda nada.
- Guardado válido: 303 a `/backlog`, con el filtro de R12 si lo había, y "Historia modificada correctamente" una sola
  vez; si la historia queda sin Story Points, con la advertencia "Historia modificada correctamente. Falta completar
  la estimación de Story Points".
- Error de validación: 422 con el formulario, los datos enviados y el mensaje junto al campo; nada cambia.
- Historia de un Sprint finalizado, al enviar: 409 con "La historia pertenece a un Sprint finalizado; no se puede
  editar"; nada cambia. Al abrir la edición: 303 a `/backlog` (con el filtro de R12) con esa misma advertencia.
- Historia inexistente, de otro proyecto o identificador inválido: página 404 de HU-20a; nada cambia.
- Falla al guardar: 500 con el formulario, los datos enviados y "La operación no pudo completarse"; nada cambia.
- Falla al leer la lista o el formulario: página de error de HU-20a (500 o 503).
- Sin proyecto activo: redirección 303 de HU-20b a elegir proyecto, con regreso a la pantalla pedida.

### Reglas de negocio

- R1: la lista se ordena por prioridad (Alta, Media, Baja); a igual prioridad, por fecha de creación (la más antigua
  primero) y, si coincide, por identificador.
- R2: el filtro acepta "Pendiente", "En Sprint", "En progreso", "Hecho" y "Todos". El filtro conserva el orden de
  R1. Un estado distinto muestra "El estado no es válido" y la lista completa.
- R3: se editan el título, la descripción, la prioridad, los Story Points y los criterios de aceptación. El estado,
  las horas estimadas y el Sprint de la historia no cambian.
- R4: título y descripción son obligatorios (sin contar espacios); la prioridad es Alta, Media o Baja; los Story
  Points son de la escala 1, 2, 3, 5, 8, 13, 21, 34, 55, 89 o "Sin estimar"; hace falta al menos un criterio no
  vacío. Son las reglas de HU-02, aplicadas con la misma validación que el alta.
- R5: una historia que pertenece a un Sprint finalizado no se edita. Se revisa dentro de la operación de guardado.
- R6: una historia de un Sprint activo (en cualquier estado: "En Sprint", "En progreso" o "Hecho") sí se edita.
- R7: los criterios se reemplazan completos al guardar, en el orden en que se envían, numerados desde 1.
- R8: los cinco datos se guardan en una sola operación; si algo se rechaza o falla, no cambia nada.
- R9: si dos personas guardan la misma historia, gana la última.
- R10: "Agregar criterio" y "Quitar" solo cambian el formulario mostrado; no guardan nada.
- R11: al abrir directamente la dirección de edición de una historia de un Sprint finalizado, no se muestra el
  formulario: se vuelve a la lista con la advertencia "La historia pertenece a un Sprint finalizado; no se puede
  editar".
- R12: la edición conserva el filtro con el que se abrió. Al guardar, al cancelar o al ser redirigido por R11, se
  vuelve a la lista con ese mismo filtro. Si el filtro llevado es inválido o no hay filtro, se vuelve a la lista
  completa, sin mensaje de error. Como la edición no cambia el estado (R3), la historia editada sigue en la lista
  filtrada.

### Restricciones

- Sin migración nueva y sin editar migraciones existentes; sin dependencias nuevas.
- Se extienden el dominio de HU-02 y los archivos de HU-22; las demás historias no se tocan.
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript (HU-20a): el filtro es un formulario `GET` y "Agregar criterio" y "Quitar"
  son botones de envío. La protección contra envíos de otro origen es el middleware global de HU-20b (los
  formularios no llevan token).
- Los mensajes nuevos se definen en un único lugar, junto a los de HU-22.

### Condiciones de error

| Condición | Mensaje | Campo | Respuesta |
|---|---|---|---|
| Estado inválido en la dirección | "El estado no es válido" (y la lista completa) | filtro | 200 |
| Filtro sin resultados | "No hay historias con ese estado" | — | 200 |
| Título vacío o solo espacios | "el título es obligatorio" (HU-02) | `titulo` | 422 |
| Descripción vacía o solo espacios | "la descripción es obligatoria" (HU-02) | `descripcion` | 422 |
| Ningún criterio no vacío | "agregá al menos un criterio de aceptación" (HU-02) | `criterio` | 422 |
| Prioridad distinta de Alta, Media o Baja | "la prioridad debe ser Alta, Media o Baja" (HU-02, como en HU-22) | `prioridad` | 422 |
| Story Points no numéricos o fuera de la escala | "Los Story Points no son un valor válido" (HU-22) | `story_points` | 422 |
| Enviar la edición de una historia de un Sprint finalizado | "La historia pertenece a un Sprint finalizado; no se puede editar" | — | 409 |
| Abrir la edición de una historia de un Sprint finalizado | La misma advertencia, en la lista | — | 303 a `/backlog` |
| Historia inexistente, de otro proyecto o identificador inválido | Página 404 de HU-20a | — | 404 |
| Falla al guardar | "La operación no pudo completarse" | — | 500 |
| Falla al leer la lista o el formulario | Página de error de HU-20a | — | 500 o 503 |
| Sin proyecto activo | Mensaje de HU-20b | — | 303 |

## Escenarios BDD

Los escenarios del Issue que combinan casos con "o" se escriben como escenarios simples, uno por caso (2a/2b,
7a/7b/7c/7d, 8a/8b, 10a/10b/10c, 11a/11b, 12a/12b, 13a/13b/13c), sin "Esquema del escenario". El escenario 12c no
está en el Issue: sale de la primera respuesta de Clarifications. El 7e tampoco: sale de la segunda.

### Bloque A – Listado y filtro

**Escenario 1: Listado ordenado**
- **Dado** un proyecto con las historias «Login» (Media, creada primero), «Pagos» (Alta), «Reportes» (Baja) y
  «Perfil» (Media, creada después de «Login»)
- **Cuando** abro el Product Backlog
- **Entonces** veo «Pagos», «Login», «Perfil» y «Reportes», en ese orden

**Escenario 2a: Filtro por estado**
- **Dado** un Backlog con historias "Pendiente", "En Sprint" y "Hecho"
- **Cuando** filtro por "Pendiente"
- **Entonces** solo veo las historias "Pendiente", en el mismo orden que en la lista completa

**Escenario 2b: Volver a ver todas**
- **Dado** que filtré el Backlog por "Pendiente"
- **Cuando** elijo "Todos"
- **Entonces** veo todas las historias del proyecto en el orden de la lista completa

**Escenario 3: Filtro sin resultados**
- **Dado** un Backlog sin historias "Hecho"
- **Cuando** filtro por "Hecho"
- **Entonces** veo "No hay historias con ese estado" y no veo un error

**Escenario 4: Filtro con un estado inválido**
- **Dado** la dirección `/backlog?estado=Archivada`
- **Cuando** abro el Backlog
- **Entonces** veo "El estado no es válido" y la lista completa

**Escenario 5: Backlog vacío**
- **Dado** un proyecto sin historias
- **Cuando** abro el Backlog
- **Entonces** veo "Aún no hay historias" y el formulario para crear la primera

### Bloque B – Edición

**Escenario 6: Acceso a la edición**
- **Dado** la historia «Login» con descripción, prioridad Media, 5 Story Points y los criterios «Acepta email» y
  «Rechaza clave vacía»
- **Cuando** presiono "Editar" en «Login»
- **Entonces** veo su formulario con ese título, esa descripción, la prioridad Media, 5 Story Points y los dos
  criterios, en ese orden

**Escenario 7a: Edición exitosa del título**
- **Dado** la historia «Login» en estado "En Sprint" con 4 horas estimadas
- **Cuando** cambio el título a «Inicio de sesión» y guardo
- **Entonces** veo "Historia modificada correctamente", la lista muestra «Inicio de sesión», y la historia sigue
  "En Sprint" con 4 horas estimadas

**Escenario 7b: Edición exitosa de la descripción**
- **Dado** la historia «Login»
- **Cuando** cambio la descripción y guardo
- **Entonces** veo "Historia modificada correctamente" y al abrir su edición veo la descripción nueva, sin cambios en
  su estado ni en sus horas estimadas

**Escenario 7c: Edición exitosa de la prioridad**
- **Dado** la historia «Reportes» con prioridad Baja
- **Cuando** cambio la prioridad a Alta y guardo
- **Entonces** veo "Historia modificada correctamente" y la lista muestra «Reportes» con prioridad Alta, ubicada
  entre las de prioridad Alta según su antigüedad

**Escenario 7d: Edición exitosa de los Story Points**
- **Dado** la historia «Login» con 5 Story Points
- **Cuando** elijo 8 Story Points y guardo
- **Entonces** veo "Historia modificada correctamente" y la lista muestra 8 Story Points en «Login», sin cambios en
  su estado ni en sus horas estimadas

**Escenario 7e: Volver a la lista filtrada después de guardar**
- **Dado** que filtré el Backlog por "Pendiente" y presioné "Editar" en «Login», que está "Pendiente"
- **Cuando** cambio el título a «Inicio de sesión» y guardo
- **Entonces** vuelvo a la lista filtrada por "Pendiente", veo "Historia modificada correctamente" e «Inicio de
  sesión» aparece en ella

**Escenario 8a: Agregar un criterio al editar**
- **Dado** el formulario de edición de «Login» con dos criterios y un título que cambié sin guardar
- **Cuando** presiono "Agregar criterio"
- **Entonces** el formulario se vuelve a mostrar con un tercer campo de criterio vacío, conserva el título que escribí
  y la historia no cambia

**Escenario 8b: Quitar un criterio al editar**
- **Dado** el formulario de edición de «Login» con los criterios «Acepta email» y «Rechaza clave vacía»
- **Cuando** presiono "Quitar" en «Acepta email»
- **Entonces** el formulario se vuelve a mostrar solo con «Rechaza clave vacía», conserva lo ya escrito y la historia
  no cambia

**Escenario 9: Quitar los Story Points**
- **Dado** la historia «Login» con 5 Story Points
- **Cuando** elijo "Sin estimar" y guardo
- **Entonces** se guarda, veo "Historia modificada correctamente. Falta completar la estimación de Story Points" y
  la lista muestra «Login» como "Sin estimar"

### Bloque C – Validaciones

**Escenario 10a: Título vacío**
- **Dado** la historia «Login»
- **Cuando** borro el título y guardo
- **Entonces** veo "el título es obligatorio" junto al título, los datos que escribí se conservan y la historia no
  cambia

**Escenario 10b: Descripción vacía**
- **Dado** la historia «Login»
- **Cuando** dejo la descripción con solo espacios y guardo
- **Entonces** veo "la descripción es obligatoria" junto a la descripción, los datos que escribí se conservan y la
  historia no cambia

**Escenario 10c: Sin criterios de aceptación**
- **Dado** la historia «Login» con dos criterios
- **Cuando** vacío los dos criterios y guardo
- **Entonces** veo "agregá al menos un criterio de aceptación" junto a los criterios, los datos que escribí se
  conservan y la historia conserva sus dos criterios

**Escenario 11a: Prioridad inválida**
- **Dado** un envío del formulario de edición de «Login» con la prioridad "Urgente"
- **Cuando** se procesa
- **Entonces** veo "la prioridad debe ser Alta, Media o Baja" junto a la prioridad, los datos se conservan y la
  historia no cambia

**Escenario 11b: Story Points fuera de la escala**
- **Dado** un envío del formulario de edición de «Login» con 4 Story Points
- **Cuando** se procesa
- **Entonces** veo "Los Story Points no son un valor válido" junto a los Story Points, los datos se conservan y la
  historia no cambia

### Bloque D – Reglas, inexistencia y fallas

**Escenario 12a: "Editar" deshabilitado para una historia de un Sprint finalizado**
- **Dado** la historia «Pagos» en estado "Hecho", que pertenece al Sprint 1 finalizado
- **Cuando** miro el Backlog
- **Entonces** "Editar" aparece deshabilitado en «Pagos» con el motivo "Sprint finalizado"

**Escenario 12b: Envío bloqueado para una historia de un Sprint finalizado**
- **Dado** la historia «Pagos» del Sprint 1 finalizado
- **Cuando** se envía igualmente un cambio de su título
- **Entonces** veo "La historia pertenece a un Sprint finalizado; no se puede editar" y la historia no cambia

**Escenario 12c: Abrir la edición de una historia de un Sprint finalizado**
- **Dado** la historia «Pagos» del Sprint 1 finalizado
- **Cuando** abro directamente la dirección de edición de «Pagos»
- **Entonces** vuelvo a la lista del Backlog con la advertencia "La historia pertenece a un Sprint finalizado; no se
  puede editar" y no veo el formulario

**Escenario 13a: Abrir la edición de una historia inexistente**
- **Dado** que no existe la historia 999999
- **Cuando** abro `/backlog/999999/editar`
- **Entonces** veo la página 404 de HU-20a y no cambia nada

**Escenario 13b: Enviar la edición con un identificador no numérico**
- **Dado** el identificador "abc"
- **Cuando** envío el formulario a `/backlog/abc/editar`
- **Entonces** veo la página 404 de HU-20a y no cambia nada

**Escenario 13c: Enviar la edición de una historia de otro proyecto**
- **Dado** que mi proyecto activo es «Alfa» y la historia «Facturas» pertenece a «Beta»
- **Cuando** envío un cambio a la edición de «Facturas»
- **Entonces** veo la página 404 de HU-20a y «Facturas» no cambia

**Escenario 14: Edición sin proyecto seleccionado**
- **Dado** que no hay proyecto seleccionado
- **Cuando** intento abrir la edición de una historia
- **Entonces** se me pide elegir un proyecto y, al elegirlo, vuelvo a la edición de esa historia

**Escenario 15: Error al guardar**
- **Dado** que completé la edición de «Login» con datos válidos y el guardado falla
- **Cuando** guardo
- **Entonces** veo "La operación no pudo completarse", los datos que escribí se conservan, no veo mensaje de éxito y
  la historia conserva sus datos originales

### Casos límite

- El identificador de la dirección vacío, no numérico, negativo, cero, con ceros a la izquierda o mayor que el
  máximo se trata como inexistente (404) sin consultar la base, tanto al abrir como al enviar.
- El 404 se decide antes de validar el formulario: un envío inválido a una historia inexistente o de otro proyecto
  responde 404, no 422.
- Si el proyecto activo cambia entre abrir y enviar el formulario, la historia es de otro proyecto y el envío
  responde 404.
- El filtro distingue mayúsculas y minúsculas: "pendiente" no es un estado válido. "Todos" y la ausencia de
  `estado` muestran la lista completa. Si `estado` viene repetido, cuenta el primero.
- "No hay historias con ese estado" solo aparece si el proyecto tiene historias; un proyecto sin historias muestra
  "Aún no hay historias" con cualquier filtro.
- Cambiar la prioridad mueve la historia a su nuevo grupo de prioridad; dentro del grupo conserva su lugar por
  antigüedad (la fecha de creación no cambia al editar).
- Los errores de formato (Story Points no numéricos) y la falta de criterios se muestran antes de aplicar las reglas
  de HU-02, que se revisan una por vez en este orden: título, descripción, prioridad, criterios y Story Points (el
  mismo orden del alta).
- Los criterios vacíos o con solo espacios se descartan; el resto se guarda recortado y numerado desde 1.
- "Quitar" con un solo campo de criterio no lo quita (como en HU-22); un `quitar_criterio:N` con N fuera de rango no
  cambia nada.
- Una historia "Hecho" de un Sprint activo se edita; si ese Sprint se cierra (HU-15) mientras se guarda, las dos
  operaciones se aplican una después de la otra: o se guarda antes del cierre, o se rechaza con 409.
- Una historia que vuelve al Backlog al cerrarse su Sprint (HU-15 la deja "Pendiente" y sin Sprint) se puede editar.
- Dos guardados simultáneos de la misma historia: se aplican uno después del otro y queda el último completo,
  criterios incluidos.
- El título, la descripción y los criterios se muestran escapados (por ejemplo, `<script>` se ve como texto).
- Recargar la lista después de guardar no reenvía el formulario ni repite el mensaje (patrón de HU-20b).
- El filtro que lleva la edición se valida como el de la lista: "pendiente", "Archivada" o un valor vacío vuelven a
  `/backlog` sin filtro y sin mensaje. Nunca se redirige a una dirección que no sea `/backlog`.
- Las horas estimadas no aparecen como campo en el formulario de edición; un campo `horas_estimadas` enviado se
  ignora.

## Requisitos funcionales

- **FR-001**: La lista del Backlog MUST ordenar las historias del proyecto activo por prioridad (Alta, Media, Baja)
  y, a igual prioridad, por fecha de creación ascendente y luego por identificador ascendente.
- **FR-002**: La lista MUST ofrecer un filtro por estado con las opciones "Todos", "Pendiente", "En Sprint", "En
  progreso" y "Hecho", que funcione sin JavaScript; al filtrar, MUST mostrar solo las historias de ese estado en el
  orden de FR-001.
- **FR-003**: "Todos" o la ausencia del estado MUST mostrar la lista completa.
- **FR-004**: Si el filtro no tiene resultados y el proyecto tiene historias, MUST mostrarse "No hay historias con ese
  estado" con respuesta 200.
- **FR-005**: Un estado inválido en la dirección MUST mostrar "El estado no es válido" y la lista completa, con
  respuesta 200.
- **FR-006**: Si el proyecto no tiene historias, MUST mostrarse "Aún no hay historias" y el formulario de alta de
  HU-22.
- **FR-007**: Cada historia de la lista MUST tener un enlace "Editar" a `/backlog/{id}/editar`; si la historia
  pertenece a un Sprint finalizado, "Editar" MUST aparecer deshabilitado con el motivo "Sprint finalizado".
- **FR-008**: `GET /backlog/{id}/editar` MUST mostrar el formulario con el título, la descripción, la prioridad, los
  Story Points y los criterios actuales, en su orden.
- **FR-009**: "Agregar criterio" y "Quitar" MUST volver a mostrar el formulario con el cambio, conservar lo escrito y
  MUST NOT guardar nada.
- **FR-010**: Un guardado válido MUST actualizar el título, la descripción, la prioridad, los Story Points y los
  criterios, responder 303 a `/backlog` y mostrar una sola vez "Historia modificada correctamente"; la lista MUST
  mostrar los valores nuevos.
- **FR-011**: Si la historia queda sin Story Points, el mensaje MUST ser la advertencia "Historia modificada
  correctamente. Falta completar la estimación de Story Points".
- **FR-012**: El guardado MUST NOT cambiar el estado, las horas estimadas, el Sprint ni la fecha de creación de la
  historia.
- **FR-013**: Título o descripción vacíos, sin criterios no vacíos, prioridad inválida o Story Points inválidos MUST
  responder 422 con el mensaje de HU-02 o HU-22 junto al campo, aplicando la misma validación que el alta.
- **FR-014**: Una historia de un Sprint finalizado MUST NOT editarse: el guardado MUST responder 409 con "La historia
  pertenece a un Sprint finalizado; no se puede editar", revisado dentro de la operación de guardado. Abrir su
  formulario MUST redirigir con 303 a `/backlog` con esa misma advertencia, una sola vez, sin mostrar el formulario.
- **FR-015**: Una historia de un Sprint activo MUST poder editarse, sea cual sea su estado.
- **FR-016**: Un identificador inválido, inexistente o de otra historia que no es del proyecto activo MUST mostrar la
  página 404 de HU-20a, al abrir y al enviar, antes de validar el formulario, y MUST NOT modificar datos.
- **FR-017**: Sin proyecto activo MUST aplicarse la redirección de HU-20b; al elegir el proyecto, MUST volverse a la
  edición pedida.
- **FR-018**: Si falla el guardado, MUST responder 500 con el formulario, los datos enviados y "La operación no pudo
  completarse", sin mensaje de éxito y sin cambios parciales.
- **FR-019**: Si falla la lectura de la lista o del formulario, MUST mostrarse la página de error de HU-20a.
- **FR-020**: Ante cualquier rechazo, el formulario MUST mostrar los datos enviados y el mensaje junto a su campo.
- **FR-021**: El guardado MUST reemplazar los criterios completos, numerados desde 1 en el orden enviado, en la misma
  operación que los demás datos; dos guardados simultáneos MUST aplicarse uno después del otro (última edición
  gana).
- **FR-022**: El título, la descripción y los criterios MUST mostrarse escapados; las respuestas de error MUST NOT
  incluir texto de consultas ni detalles internos.
- **FR-023**: HU-13 MUST NOT agregar migraciones y MUST definir sus mensajes nuevos en un único lugar, junto a los de
  HU-22.
- **FR-024**: El enlace "Editar" MUST llevar el filtro de la lista. El formulario MUST conservarlo en un campo
  oculto, incluso ante un rechazo. Al guardar, al cancelar y en la redirección de FR-014, MUST volverse a
  `/backlog?estado=…` con ese filtro. Un filtro inválido o ausente MUST volver a `/backlog`, sin mensaje de error.

## Entidades clave

- **Historia de usuario** (HU-02): título, descripción, prioridad, Story Points, criterios de aceptación, estado,
  horas estimadas, fecha de creación. HU-13 modifica los cinco primeros.
- **Criterio de aceptación** (HU-02): texto con posición dentro de la historia.
- **Sprint** (HU-04): "Activo" o "Finalizado"; una historia pertenece como mucho a un Sprint.

## Criterios de éxito

- **SC-001**: En el 100% de las pruebas, la lista y cualquier filtro respetan el orden de FR-001.
- **SC-002**: El 100% de los guardados válidos se ve en la lista y en el formulario de edición, sin cambios en el
  estado, las horas estimadas, el Sprint ni la fecha de creación.
- **SC-003**: Ningún envío rechazado (validación, Sprint finalizado, historia inexistente o de otro proyecto, falla
  al guardar) cambia la historia ni sus criterios.
- **SC-004**: Ninguna historia de un Sprint finalizado cambia después del cierre del Sprint, aunque el guardado y el
  cierre ocurran al mismo tiempo.
- **SC-005**: Un Scrum Master filtra el Backlog en dos acciones (elegir el estado y filtrar) y edita una historia en
  tres (Editar, cambiar, Guardar); la lista y el formulario responden en menos de 500 ms con la base local y 200
  historias en el proyecto.
- **SC-006**: Cada escenario tiene una prueba que falló antes del código (evidencia RED) y pasa después (evidencia
  GREEN), y la cobertura de `src/backlog` queda en `cobertura.txt` con al menos 80%.

## Supuestos y decisiones

Decisiones del equipo (no se vuelven a preguntar):

1. Orden: Alta, Media, Baja; a igual prioridad, la más antigua primero (fecha de creación y luego identificador).
2. Estados del filtro: Pendiente, En Sprint, En progreso y Hecho, más "Todos". Un estado inválido en la dirección
   muestra "El estado no es válido" y la lista completa. Sin resultados: "No hay historias con ese estado".
3. Se editan título, descripción, prioridad, Story Points y criterios. No se editan las horas estimadas (HU-17), el
   estado ni el Sprint.
4. Una historia de un Sprint finalizado no se edita. En la lista, "Editar" aparece deshabilitado con el motivo
   "Sprint finalizado"; si igual se envía, responde 409 con "La historia pertenece a un Sprint finalizado; no se
   puede editar". La regla se revisa dentro de la transacción de guardado.
5. Las historias de un Sprint activo ("En Sprint", "En progreso", "Hecho") sí se pueden editar.
6. Textos: éxito "Historia modificada correctamente"; sin Story Points, advertencia "Historia modificada
   correctamente. Falta completar la estimación de Story Points". Los mensajes de título, descripción, criterios,
   prioridad y Story Points son los de HU-02 y HU-22.
7. Los criterios se reemplazan completos en una sola transacción (se reescriben las posiciones 1..n). Última edición
   gana.

Supuestos:

- Una historia "pertenece a un Sprint finalizado" si tiene asignación a un Sprint con estado "Finalizado". Hoy solo
  HU-15 (cerrar Sprint) deja Sprints en ese estado y conserva la asignación de las historias "Hecho"; las pruebas
  preparan esos Sprints directamente en la base.
- Los mensajes de HU-02 para título, descripción, prioridad y criterios están en minúscula ("el título es
  obligatorio"); HU-13 los reutiliza tal cual, como HU-22.
- La lista vacía ("Aún no hay historias" y el formulario de alta) ya la resuelve HU-22; HU-13 la conserva con el
  filtro.
- La sección "Backlog" exige proyecto activo, no integrante identificado (HU-22).
- La edición no revisa el estado guardado del proyecto (`projects.status`), a diferencia del alta de HU-02, que
  exige "Activo". Ninguna historia cambia hoy ese estado (HU-12 lo calcula con las fechas), y el proyecto ya está
  validado por el contexto de HU-20b y por la búsqueda de la historia dentro de él. Si en el futuro un proyecto puede
  dejar de estar "Activo", hay que decidir si la edición también se bloquea.

## Trazabilidad

| Escenario | Requisitos | Éxito | Prueba de handler / BDD | Prueba de dominio / integración |
|---|---|---|---|---|
| 1 Listado ordenado | FR-001 | SC-001 | `TestHU13_ListadoOrdenado`, `TestBacklogHandler_GetListsActiveProjectStoriesInPriorityOrder`, BDD "Listado ordenado" | `TestPostgreSQLStoryCatalogListsOnlyProjectStoriesInPriorityOrder`, `TestPostgreSQLStoryCatalog_OrdenPorPrioridadYAntiguedad` |
| 2a/2b Filtro por estado | FR-002, FR-003 | SC-001, SC-005 | `TestHU13_FiltroPorEstado`, BDD "Filtro por estado" | `TestStoryStatuses` |
| 3 Filtro sin resultados | FR-004 | — | `TestHU13_FiltroSinResultados` | — |
| 4 Estado inválido | FR-005 | — | `TestHU13_FiltroEstadoInvalido` | — |
| 5 Backlog vacío | FR-006 | — | `TestHU13_BacklogVacioConFiltro` | — |
| 6 Acceso a la edición | FR-007, FR-008 | SC-005 | `TestHU13_FormularioConDatosActuales` | `TestPostgresStoryForEdit_ConCriteriosEnOrden`, `TestPostgresStoryForEdit_SinCriterios` |
| 7a/7b/7c/7d Edición exitosa | FR-010, FR-012, FR-021 | SC-002 | `TestHU13_EdicionExitosa`, `TestHU13_IgnoraHorasEstimadas`, BDD "Edición exitosa" | `TestUpdateStory_DelegaYPropaga`, `TestPostgresUpdateStory_DatosYSinEfectos`, `TestPostgresUpdateStory_HechoEnSprintActivo` |
| 7e Volver a la lista filtrada | FR-024 | SC-005 | `TestHU13_VuelveALaListaFiltrada` | — |
| 8a/8b Agregar y quitar criterios | FR-009 | SC-003 | `TestHU13_AgregarYQuitarCriterio` | `TestPostgresUpdateStory_ReemplazaCriterios` |
| 9 Quitar Story Points | FR-010, FR-011 | SC-002 | `TestHU13_QuitarStoryPoints`, BDD "Quitar los Story Points" | `TestUpdateStory_DelegaYPropaga`, `TestPostgresUpdateStory_SinStoryPoints` |
| 10a/10b/10c Edición inválida | FR-013, FR-020 | SC-003 | `TestHU13_TituloODescripcionVacios`, `TestHU13_SinCriterios`, BDD "Edición inválida" | `TestValidateStoryFields`, `TestUpdateStory_Validaciones` |
| 11a/11b Prioridad o Story Points inválidos | FR-013, FR-020 | SC-003 | `TestHU13_PrioridadInvalida`, `TestHU13_StoryPointsInvalidos` | `TestValidateStoryFields`, `TestUpdateStory_Validaciones` |
| 12a/12b/12c Sprint finalizado | FR-007, FR-014, FR-015 | SC-003, SC-004 | `TestHU13_EditarDeshabilitadoSprintFinalizado`, `TestHU13_EnvioBloqueadoSprintFinalizado`, `TestHU13_AbrirEdicionSprintFinalizado`, BDD "Historia de un Sprint finalizado" | `TestPostgreSQLStoryCatalog_MarcaSprintFinalizado`, `TestPostgresStoryForEdit_SprintFinalizado`, `TestPostgresUpdateStory_SprintFinalizado`, `TestPostgresUpdateStory_EsperaAlCierre` |
| 13a/13b/13c Inexistente o de otro proyecto | FR-016 | SC-003 | `TestHU13_HistoriaInexistente`, `TestHU13_404AntesQue422` | `TestPostgresUpdateStory_InexistenteYOtroProyecto`, `TestPostgresStoryForEdit_Inexistente`, `TestPostgresStoryForEdit_OtroProyecto` |
| 14 Sin proyecto | FR-017 | — | `TestHU13_EdicionSinProyecto` | — |
| 15 Error al guardar | FR-018, FR-022 | SC-003 | `TestHU13_ErrorAlGuardar`, BDD "Error al guardar" | `TestPostgresUpdateStory_FalloNoDejaCambios` |
| (HU-20a) Error al leer | FR-019 | — | `TestHU13_ErrorAlLeer` | — |
| (HU-20b) Texto escapado | FR-022 | — | `TestHU13_TextoEscapado` | — |
| (HU-20b) Recarga sin reenviar | FR-010 | SC-002 | `TestHU13_RecargaSinReenviar` | — |
| Concurrencia (última gana, asignación cambiada) | FR-021 | — | — | `TestPostgresUpdateStory_ConcurrenteUltimaGana`, `TestPostgresUpdateStory_AsignacionCambiaReintenta`, `TestPostgresUpdateStory_AsignacionCambiaDosVeces`, `TestPostgresUpdateStory_SinCambios` |
| Rendimiento | — | SC-005 | `TestHU13_Rendimiento` | — |
| Sin efectos fuera de alcance | FR-012, FR-023 | — | — | `TestPostgresUpdateStory_DatosYSinEfectos` |
