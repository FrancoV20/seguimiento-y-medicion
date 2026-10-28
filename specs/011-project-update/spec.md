# Especificación: HU-11 – Modificación de Proyecto (web)

**Rama**: `HU-11`
**Creada**: 2026-10-27
**Estado**: Borrador
**Constitución**: v1.1.0
**Depende de**: HU-01, FIX-HU-01, HU-03, HU-04, HU-20a, HU-20b, HU-21
**Prioridad**: Media (Should Have) · **Valor**: 70 · **Estimación**: 5 SP (sugerida)

**Entrada**: "HU-11 / Modificación de Proyecto (web). Como Usuario, quiero modificar el nombre, las fechas y los
integrantes de un proyecto existente desde la web, para mantener sus datos actualizados."

## Historia de usuario

Como Usuario, quiero modificar el nombre, las fechas y los integrantes de un proyecto existente desde la web, para
mantener sus datos actualizados.

> **Alcance**: HU-11 agrega a "Gestión de Proyectos" (HU-21) un enlace "Modificar" en cada proyecto y un
> formulario que guarda el nombre, las fechas y los integrantes en una sola operación. Reutiliza las reglas y los
> mensajes de HU-01 (nombre obligatorio, al menos un integrante, fin posterior al inicio) y los de formato de
> HU-21. Agrega dos reglas propias: no se quita a un integrante con horas en el proyecto y, si cambian las fechas,
> ningún Sprint queda fuera de ellas.
>
> **Sin contexto**: la sección "Proyectos" no exige proyecto activo ni integrante identificado (HU-21); cualquier
> proyecto se puede modificar, sea o no el contexto de trabajo.

## Clarifications

### Session 2026-10-27

- Q: ¿Cómo se evita que un integrante cargue horas (HU-24) mientras otra persona lo está quitando del proyecto?
  → A: HU-11 quita la pertenencia antes de contar las horas, dentro de su transacción y con el proyecto
  bloqueado; si encuentra horas, deshace todo. El guardado de horas de HU-03 vuelve a comprobar la pertenencia
  dentro de su transacción y la bloquea en modo compartido. La operación que llega primero define el resultado.
- Q: Cuando se cambian las fechas del proyecto, ¿se revisan todos sus Sprints o solo los que ese cambio deja
  fuera? → A: Solo los Sprints que estaban dentro de las fechas actuales y quedan fuera de las nuevas. Los que ya
  estaban fuera no bloquean el cambio. Se revisan los Sprints "Activo" y "Finalizado".

## Sección SDD

### Intención

Permitir corregir los datos de un proyecto ya creado sin perder la coherencia con lo que depende de él: las horas
registradas por sus integrantes (HU-03) y las fechas de sus Sprints (HU-04).

### Alcance

- Enlace "Modificar" en cada fila de la lista de "Gestión de Proyectos".
- Formulario `/proyectos/{id}/modificar` con el nombre, las fechas y todos los integrantes del sistema, con los
  actuales marcados.
- Guardar nombre, fechas e integrantes juntos: todo o nada.
- Agregar integrantes que ya existen en el sistema y quitar integrantes sin horas en el proyecto.
- Rechazos: nombre vacío, fechas faltantes, con formato inválido o con fin anterior o igual al inicio, ningún
  integrante, integrante inválido, integrante con horas, Sprint fuera de las nuevas fechas, proyecto inexistente,
  falla al guardar.

### Fuera de alcance

- Crear integrantes nuevos (el formulario solo marca integrantes existentes).
- Borrar proyectos y cambiar su estado (HU-12 lo calcula a partir de las fechas).
- Modificar, mover o borrar Sprints, historias u horas para "hacer lugar" al cambio.
- Avisar a otras personas que tienen el formulario abierto (última edición gana).
- Roles y permisos (no hay roles; limitación conocida de HU-20b).

### Entradas

| Formulario | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| dirección | `{id}` | identificador numérico | sí | Proyecto a modificar |
| `POST /proyectos/{id}/modificar` | `nombre` | texto | sí | Se recortan los espacios; hasta 255 caracteres |
| `POST /proyectos/{id}/modificar` | `fecha_inicio` | fecha `AAAA-MM-DD` | sí | |
| `POST /proyectos/{id}/modificar` | `fecha_fin` | fecha `AAAA-MM-DD` | sí | Posterior a `fecha_inicio` |
| `POST /proyectos/{id}/modificar` | `integrantes` | identificadores numéricos (varios) | al menos uno | Integrantes existentes; repetidos se ignoran |

Los nombres de los campos son los de HU-21.

### Salidas esperadas

- Guardado válido (con o sin cambios): redirección 303 a `/proyectos` con "Proyecto modificado exitosamente" una
  sola vez; la lista muestra los valores nuevos.
- Error de validación (nombre, fechas, integrantes): 422 con el formulario, los datos enviados y el mensaje junto
  al campo; nada cambia.
- Integrante con horas o Sprint fuera de las fechas: 409 con el formulario, los datos enviados y el mensaje junto
  al campo; nada cambia.
- Proyecto inexistente o identificador inválido: página 404 de HU-20a; nada cambia.
- Falla al guardar: 500 con el formulario, los datos enviados y "La operación no pudo completarse"; nada cambia.
- Falla al leer el formulario o la lista: página de error de HU-20a (500 o 503).

### Reglas de negocio

- R1: el nombre es obligatorio (sin contar espacios); se aplica la regla de HU-01. El nombre se guarda sin los
  espacios del principio y del final, también si el servicio se usa sin pasar por la web.
- R2: la fecha de fin debe ser posterior a la de inicio; se aplica la única función de validación de fechas de
  HU-01 (FIX-HU-01, R2), sin copiarla.
- R3: el proyecto debe quedar con al menos un integrante (HU-01).
- R4: solo se marcan integrantes que existen en el sistema; no se crean integrantes.
- R5: no se quita a un integrante que tiene horas registradas en historias de este proyecto; las horas en otros
  proyectos no cuentan.
- R6: si cambia alguna de las dos fechas, ningún Sprint que estaba dentro de las fechas actuales del proyecto
  puede quedar fuera de las nuevas. Un Sprint está dentro de un rango si empieza el día de inicio o después y
  termina el día de fin o antes. Los Sprints que ya estaban fuera (HU-04 y HU-23 no lo impiden) no bloquean el
  cambio y siguen como están. Cuentan los Sprints "Activo" y "Finalizado". El mensaje nombra el Sprint de menor
  número entre los que el cambio deja fuera.
- R7: si las fechas no cambian, ningún Sprint puede quedar fuera por el cambio (R6), así que no se leen los Sprints.
  Es una consecuencia de R6 y no cambia el resultado: solo evita una lectura.
- R8: nombre, fechas e integrantes se guardan en una sola operación; si algo se rechaza o falla, no se guarda
  nada.
- R9: guardar sin cambios cuenta como éxito.
- R10: si dos personas guardan el mismo proyecto, gana la última; las reglas R5 y R6 se revisan siempre dentro de
  la operación de guardado, con el proyecto bloqueado.
- R11: un integrante no puede cargar horas en el proyecto después de haber sido quitado. Quitarlo y cargar horas
  al mismo tiempo se ordenan sobre su pertenencia al proyecto. Si la carga de horas llega primero, HU-11 espera a que
  termine, ve esas horas y rechaza el cambio con el mensaje de R5. Si HU-11 llega primero, la carga espera, ve que
  el integrante ya no pertenece y la rechaza con el error que HU-03 ya tiene para un integrante que no pertenece al
  proyecto.

### Restricciones

- Sin migración nueva y sin editar migraciones existentes; sin dependencias nuevas.
- Se extienden los archivos de HU-21 y el dominio de HU-01. De HU-03 solo cambia el guardado de horas, que vuelve
  a comprobar y bloquear la pertenencia dentro de su transacción (R11), sin mensajes nuevos y con aviso a su
  responsable. Las demás historias no se tocan.
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript (HU-20a); la protección contra envíos de otro origen es el middleware global
  de HU-20b (los formularios no llevan token).
- Los mensajes nuevos se definen en un único lugar, junto a los de HU-21.

### Condiciones de error

| Condición | Mensaje | Campo | Respuesta |
|---|---|---|---|
| Nombre vacío o solo espacios | Mensaje de HU-01: "project name cannot be empty" | `nombre` | 422 |
| Nombre de más de 255 caracteres | "El nombre no puede superar los 255 caracteres" (HU-21) | `nombre` | 422 |
| Falta la fecha de inicio | "Falta la fecha de inicio" | `fecha_inicio` | 422 |
| Falta la fecha de fin | "Falta la fecha de fin" | `fecha_fin` | 422 |
| Fecha fuera del formato `AAAA-MM-DD` | "La fecha no tiene un formato válido" | esa fecha | 422 |
| Fin anterior o igual al inicio | Mensaje de HU-01: "La fecha de finalización debe ser posterior a la fecha de inicio" | `fecha_fin` | 422 |
| Ningún integrante marcado | Mensaje de HU-01: "the member list is required" | `integrantes` | 422 |
| Integrante inexistente, no numérico o menor o igual a cero | "El integrante seleccionado no es válido" | `integrantes` | 422 |
| Quitar un integrante con horas en el proyecto | "No se puede quitar a [nombre]: tiene horas registradas en este proyecto" | `integrantes` | 409 |
| Un Sprint queda fuera de las nuevas fechas | "El Sprint N queda fuera de las nuevas fechas del proyecto" | la fecha que lo causa | 409 |
| Proyecto inexistente o identificador inválido | Página 404 de HU-20a | — | 404 |
| Falla al guardar | "La operación no pudo completarse" | — | 500 |
| Falla al leer el formulario o la lista | Página de error de HU-20a | — | 500 o 503 |

## Escenarios BDD

Los escenarios del Issue que combinan varios casos con "o" se escriben como escenarios simples, uno por caso
(2a/2b, 4a/4b, 6a/6b/6c, 12a/12b, 13a/13b, 14a/14b), sin "Esquema del escenario". El escenario 11b no está en el
Issue: sale de la segunda respuesta de Clarifications.

### Bloque A – Acceso y datos del proyecto

**Escenario 1: Acceso desde la lista**
- **Dado** que en "Gestión de Proyectos" está el proyecto «Alfa» con fechas 2026-03-01 y 2026-06-30 y los
  integrantes Ana y Beto
- **Cuando** miro la lista y sigo el enlace "Modificar" de «Alfa»
- **Entonces** veo su formulario con el nombre «Alfa», las fechas 2026-03-01 y 2026-06-30, y Ana y Beto marcados

**Escenario 2a: Modificación exitosa del nombre**
- **Dado** el proyecto «Alfa»
- **Cuando** cambio el nombre a «Alfa 2» y guardo
- **Entonces** veo "Proyecto modificado exitosamente", vuelvo a la lista y el proyecto se muestra como «Alfa 2»

**Escenario 2b: Modificación exitosa de las fechas**
- **Dado** el proyecto «Alfa» con fechas 2026-03-01 y 2026-06-30 y sin Sprints
- **Cuando** cambio las fechas a 2026-04-01 y 2026-07-31 y guardo
- **Entonces** veo "Proyecto modificado exitosamente", vuelvo a la lista y el proyecto muestra 01/04/2026 y
  31/07/2026

**Escenario 3: Nombre visible en la estructura común**
- **Dado** que «Alfa» es mi proyecto de trabajo
- **Cuando** cambio su nombre a «Alfa 2» y guardo
- **Entonces** la estructura común muestra «Alfa 2» como proyecto de trabajo sin que lo elija otra vez

**Escenario 4a: Nombre vacío**
- **Dado** el proyecto «Alfa»
- **Cuando** borro el nombre y guardo
- **Entonces** veo "project name cannot be empty" junto al nombre, el resto de los datos que envié se conservan y
  el proyecto sigue llamándose «Alfa»

**Escenario 4b: Nombre con solo espacios**
- **Dado** el proyecto «Alfa»
- **Cuando** escribo solo espacios en el nombre y guardo
- **Entonces** veo "project name cannot be empty" junto al nombre, el resto de los datos que envié se conservan y
  el proyecto sigue llamándose «Alfa»

**Escenario 5: Fechas inválidas al modificar**
- **Dado** el proyecto «Alfa» con fechas 2026-03-01 y 2026-06-30
- **Cuando** pongo como fecha de fin 2026-03-01 y guardo
- **Entonces** veo "La fecha de finalización debe ser posterior a la fecha de inicio" junto a la fecha de fin, los
  datos que envié se conservan y las fechas del proyecto no cambian

**Escenario 6a: Falta la fecha de inicio**
- **Dado** el proyecto «Alfa»
- **Cuando** dejo vacía la fecha de inicio y guardo
- **Entonces** veo "Falta la fecha de inicio" junto a la fecha de inicio, los datos que envié se conservan y el
  proyecto no cambia

**Escenario 6b: Falta la fecha de fin**
- **Dado** el proyecto «Alfa»
- **Cuando** dejo vacía la fecha de fin y guardo
- **Entonces** veo "Falta la fecha de fin" junto a la fecha de fin, los datos que envié se conservan y el
  proyecto no cambia

**Escenario 6c: Fecha con formato inválido**
- **Dado** el proyecto «Alfa»
- **Cuando** envío la fecha de inicio "01/03/2026" y guardo
- **Entonces** veo "La fecha no tiene un formato válido" junto a la fecha de inicio, los datos que envié se
  conservan y el proyecto no cambia

### Bloque B – Integrantes

**Escenario 7: Agregar integrante**
- **Dado** el proyecto «Alfa» con Ana y el integrante Carla del sistema, que no pertenece a «Alfa»
- **Cuando** marco a Carla y guardo
- **Entonces** Carla pertenece a «Alfa», aparece en el selector de identificación de HU-20b y puede cargar
  esfuerzo en una historia de «Alfa»

**Escenario 8: Quitar un integrante sin horas**
- **Dado** el proyecto «Alfa» con Ana y Beto, y Beto sin horas en «Alfa» pero con horas en el proyecto «Beta»
- **Cuando** desmarco a Beto y guardo
- **Entonces** Beto deja de pertenecer a «Alfa», ya no aparece en el selector de identificación de «Alfa» y sus
  horas en «Beta» no cambian

**Escenario 9: Quitar el último integrante**
- **Dado** el proyecto «Alfa» con solo Ana
- **Cuando** desmarco a Ana y guardo
- **Entonces** veo "the member list is required" junto al campo de integrantes, los datos que envié se conservan
  y Ana sigue en «Alfa»

**Escenario 10: Quitar un integrante con horas registradas**
- **Dado** el proyecto «Alfa» con Ana y Beto, y Beto con horas en una historia de «Alfa»
- **Cuando** desmarco a Beto, cambio el nombre a «Alfa 2» y guardo
- **Entonces** veo "No se puede quitar a Beto: tiene horas registradas en este proyecto" junto al campo de
  integrantes, Beto sigue en «Alfa» y el nombre sigue siendo «Alfa»

**Escenario 13a: Integrante inexistente**
- **Dado** un envío del formulario de «Alfa» con un integrante que no existe en el sistema
- **Cuando** se procesa
- **Entonces** veo "El integrante seleccionado no es válido" junto al campo de integrantes, los datos que envié se
  conservan y el proyecto no cambia

**Escenario 13b: Integrante no numérico**
- **Dado** un envío del formulario de «Alfa» con el integrante "abc"
- **Cuando** se procesa
- **Entonces** veo "El integrante seleccionado no es válido" junto al campo de integrantes, los datos que envié se
  conservan y el proyecto no cambia

**Escenario 15: Integrante identificado que fue quitado**
- **Dado** que estoy identificado como Beto en «Alfa» y otra persona quitó a Beto de «Alfa»
- **Cuando** navego por la aplicación
- **Entonces** mi identificación se limpia, veo la advertencia de HU-20b y se me pide elegir otro integrante al
  entrar a una sección que lo exige

### Bloque C – Fechas y Sprints

**Escenario 11a: Fechas que dejan Sprints fuera**
- **Dado** el proyecto «Alfa» con fechas 2026-03-01 y 2026-06-30 y el Sprint 2 del 2026-06-01 al 2026-06-14
- **Cuando** cambio la fecha de fin a 2026-06-10 y guardo
- **Entonces** veo "El Sprint 2 queda fuera de las nuevas fechas del proyecto" junto a la fecha de fin, los datos
  que envié se conservan y las fechas del proyecto no cambian

**Escenario 11b: Un Sprint que ya estaba fuera no bloquea el cambio**
- **Dado** el proyecto «Alfa» con fechas 2026-03-01 y 2026-06-30 y el Sprint 1 del 2026-02-20 al 2026-03-05, que
  ya empieza antes del proyecto
- **Cuando** cambio la fecha de fin a 2026-07-31 y guardo
- **Entonces** veo "Proyecto modificado exitosamente", el proyecto termina el 31/07/2026 y el Sprint 1 no cambia

**Escenario 12a: Cambiar solo el nombre con Sprints creados**
- **Dado** el proyecto «Alfa» con Sprints creados
- **Cuando** cambio solo el nombre y guardo
- **Entonces** veo "Proyecto modificado exitosamente" y el nombre cambia sin que se revisen las fechas de los
  Sprints

**Escenario 12b: Cambiar solo los integrantes con Sprints creados**
- **Dado** el proyecto «Alfa» con Sprints creados
- **Cuando** marco a Carla, sin cambiar las fechas, y guardo
- **Entonces** veo "Proyecto modificado exitosamente" y Carla pertenece a «Alfa» sin que se revisen las fechas de
  los Sprints

### Bloque D – Proyecto inexistente y fallas

**Escenario 14a: Abrir el formulario de un proyecto inexistente**
- **Dado** que no existe un proyecto con el identificador 999999
- **Cuando** abro `/proyectos/999999/modificar`
- **Entonces** veo la página 404 de HU-20a y no cambia nada

**Escenario 14b: Enviar el formulario con un identificador no numérico**
- **Dado** el identificador de proyecto "abc"
- **Cuando** envío el formulario a `/proyectos/abc/modificar`
- **Entonces** veo la página 404 de HU-20a y no cambia nada

**Escenario 16: Error al guardar**
- **Dado** que completé el formulario de «Alfa» con datos válidos y el guardado falla
- **Cuando** guardo
- **Entonces** veo "La operación no pudo completarse", los datos que envié se conservan, no veo mensaje de éxito y
  no cambian ni el nombre, ni las fechas, ni los integrantes

### Casos límite

- El identificador de la dirección vacío, no numérico, negativo, cero, con ceros a la izquierda o mayor que el
  máximo se trata como inexistente (404) sin consultar la base, tanto al abrir como al enviar.
- Si el proyecto existe al abrir el formulario pero se borra antes de guardar, el envío responde 404 y no crea
  nada.
- El 404 se decide antes de validar el formulario: un envío inválido a un proyecto inexistente responde 404, no 422.
- Los errores de formato (fechas faltantes o con formato inválido, integrante no numérico, nombre de más de 255
  caracteres) se muestran todos juntos, como en HU-21. Las reglas de HU-01 se revisan después, una por vez:
  nombre, integrantes y fechas. Las reglas de guardado se revisan al final, una por vez: integrante inexistente,
  integrante con horas y Sprints.
- Si se quitan varios integrantes con horas, el mensaje nombra al primero por nombre (y por identificador si hay
  nombres iguales).
- Un integrante repetido en el envío se cuenta una sola vez.
- Marcar a un integrante que ya pertenece y desmarcar a otro sin horas en el mismo envío se guarda junto.
- Fecha de fin un día posterior a la de inicio: se acepta (FIX-HU-01).
- Un Sprint que empieza el mismo día que el proyecto o termina el mismo día que el proyecto queda dentro.
- Si el Sprint afectado empieza antes del nuevo inicio, el mensaje va junto a la fecha de inicio; si termina
  después del nuevo fin, junto a la fecha de fin. Si queda fuera por los dos lados, el mensaje va junto a la fecha
  de inicio.
- Los Sprints "Activo" y "Finalizado" cuentan igual para la regla de fechas.
- Ampliar las fechas nunca deja fuera un Sprint que antes estaba dentro, así que nunca se rechaza por Sprints.
- Un Sprint que ya estaba fuera de las fechas actuales (por ejemplo, empieza antes del inicio) no bloquea el
  cambio, aunque las nuevas fechas también lo dejen fuera.
- Si hay un Sprint que ya estaba fuera y otro que el cambio deja fuera, el mensaje nombra al segundo.
- Cambiar solo el formato de la fecha (misma fecha) no es un cambio de fechas.
- Dos guardados simultáneos del mismo proyecto: se aplican uno después del otro y queda el último; cada uno revisa
  las reglas con los datos que encuentra al guardar.
- Quitar a Beto mientras Beto carga horas en «Alfa»: termina una sola de las dos operaciones. O queda la carga y
  HU-11 responde 409 "No se puede quitar a Beto…", o queda la baja y la carga recibe el error de HU-03 de
  integrante que no pertenece. Nunca quedan las dos.
- Crear un Sprint (HU-04) mientras se acortan las fechas: las dos operaciones bloquean el proyecto y se aplican
  una después de la otra.
- El nombre del proyecto y de los integrantes se muestra escapado (por ejemplo, `<script>` se ve como texto).
- Recargar la lista después de guardar no reenvía el formulario ni repite el mensaje (patrón de HU-20b).
- El proyecto se puede modificar sin proyecto activo ni integrante identificado.

## Requisitos funcionales

- **FR-001**: Cada fila de la lista de "Gestión de Proyectos" MUST tener un enlace "Modificar" a
  `/proyectos/{id}/modificar`.
- **FR-002**: `GET /proyectos/{id}/modificar` MUST mostrar el formulario con el nombre actual, las fechas actuales
  en formato `AAAA-MM-DD` y todos los integrantes del sistema, con los del proyecto marcados.
- **FR-003**: Un guardado válido MUST responder 303 a `/proyectos`, mostrar una sola vez "Proyecto modificado
  exitosamente" y la lista MUST mostrar el nombre, las fechas y la cantidad de integrantes nuevos.
- **FR-004**: Nombre, fechas e integrantes MUST guardarse en una sola operación: si cualquier regla rechaza el
  envío o el guardado falla, MUST NOT cambiar ninguno de los tres.
- **FR-005**: Un guardado sin cambios MUST tratarse como éxito (FR-003).
- **FR-006**: Un nombre vacío o con solo espacios MUST responder 422 con el mensaje de HU-01 junto al nombre; un
  nombre de más de 255 caracteres MUST responder 422 con el mensaje de HU-21.
- **FR-007**: Una fecha vacía o fuera del formato `AAAA-MM-DD` MUST responder 422 con "Falta la fecha de inicio",
  "Falta la fecha de fin" o "La fecha no tiene un formato válido" junto a esa fecha; una fecha de fin anterior o
  igual a la de inicio MUST responder 422 con el mensaje de HU-01 junto a la fecha de fin, aplicando la misma
  función de validación que el alta.
- **FR-008**: Sin integrantes marcados, el guardado MUST responder 422 con el mensaje de HU-01 junto al campo de
  integrantes.
- **FR-009**: Un integrante no numérico, menor o igual a cero o inexistente MUST responder 422 con "El integrante
  seleccionado no es válido" junto al campo de integrantes; MUST NOT crearse integrantes.
- **FR-010**: Quitar a un integrante con horas registradas en historias de este proyecto MUST responder 409 con
  "No se puede quitar a [nombre]: tiene horas registradas en este proyecto" junto al campo de integrantes; las horas
  en otros proyectos MUST NOT impedir quitarlo.
- **FR-011**: Un integrante quitado MUST dejar de pertenecer al proyecto y MUST NOT aparecer en el selector de
  identificación de HU-20b; un integrante agregado MUST pertenecer al proyecto, aparecer en ese selector y poder
  cargar esfuerzo (HU-24).
- **FR-012**: Si cambia alguna fecha y un Sprint que estaba dentro de las fechas actuales queda fuera de las
  nuevas (R6), el guardado MUST responder 409 con "El Sprint N queda fuera de las nuevas fechas del proyecto" junto a
  la fecha que lo causa, nombrando el de menor número entre esos Sprints. Un Sprint que ya estaba fuera MUST NOT
  bloquear el cambio.
- **FR-013**: Si las fechas no cambian, el guardado MUST aceptarse sin importar las fechas de los Sprints, y MUST
  NOT leer los Sprints del proyecto. Lo primero se ve en la pantalla (escenarios 12a y 12b). Lo segundo no cambia el
  resultado, porque R6 ya lo implica: se verifica con la prueba de `DatesChanged` y en la revisión del código.
- **FR-014**: Un identificador de proyecto inválido o inexistente MUST mostrar la página 404 de HU-20a, tanto al
  abrir como al enviar el formulario, y MUST NOT modificar datos.
- **FR-015**: Si falla el guardado, el sistema MUST responder 500 con el formulario, los datos enviados y "La
  operación no pudo completarse", y MUST NOT mostrar un mensaje de éxito ni dejar cambios parciales.
- **FR-016**: Si falla la lectura del formulario o de la lista, el sistema MUST mostrar la página de error de
  HU-20a.
- **FR-017**: Ante cualquier rechazo, el formulario MUST mostrar los datos enviados (nombre, fechas e integrantes
  marcados) y el mensaje junto a su campo.
- **FR-018**: Después de cambiar el nombre del proyecto de trabajo, la estructura común MUST mostrar el nombre
  nuevo en la siguiente respuesta, sin volver a elegir el proyecto.
- **FR-019**: El guardado MUST bloquear el proyecto mientras dura y MUST revisar R5 y R6 dentro de esa misma
  operación; dos guardados simultáneos MUST aplicarse uno después del otro (última edición gana).
- **FR-020**: Un integrante MUST NOT quedar con horas registradas en un proyecto del que fue quitado, aunque cargue
  horas mientras se lo quita (R11). El guardado de HU-11 MUST quitar la pertenencia antes de contar las horas, en la
  misma transacción. El guardado de horas de HU-03 MUST comprobar la pertenencia dentro de su transacción con un
  bloqueo compartido, y MUST rechazar la carga con su error actual si el integrante ya no pertenece.
- **FR-021**: Los nombres del proyecto y de los integrantes MUST mostrarse escapados; las respuestas de error MUST
  NOT incluir texto de consultas ni detalles internos.
- **FR-022**: HU-11 MUST NOT modificar Sprints, historias, horas, integrantes del sistema ni el estado guardado del
  proyecto, y MUST NOT agregar migraciones.
- **FR-023**: Los mensajes nuevos de HU-11 MUST definirse en un único lugar, junto a los de HU-21.

## Entidades clave

- **Proyecto** (HU-01): nombre, fecha de inicio, fecha de fin, integrantes. HU-11 modifica estos cuatro datos.
- **Integrante** (HU-01): existe en el sistema; pertenece o no a cada proyecto.
- **Registro de horas** (HU-03): horas de un integrante en una historia; la historia pertenece a un proyecto.
- **Sprint** (HU-04): número, fecha de inicio y fecha de fin dentro de un proyecto.

## Criterios de éxito

- **SC-001**: El 100% de los guardados válidos se ve en la lista (nombre, fechas, cantidad de integrantes) y, si es
  el proyecto de trabajo, en la estructura común de la siguiente página.
- **SC-002**: Ningún envío rechazado (validación, integrante con horas, Sprint fuera de las fechas, proyecto
  inexistente, falla al guardar) cambia el nombre, las fechas ni los integrantes.
- **SC-003**: En ningún caso queda un integrante quitado de un proyecto con horas registradas en historias de ese
  proyecto, ni siquiera si carga horas mientras se lo quita.
- **SC-004**: Después de un cambio de fechas aceptado, el 100% de los Sprints que estaban dentro de las fechas
  anteriores siguen dentro de las nuevas, y ninguna ampliación de fechas se rechaza por un Sprint.
- **SC-005**: Ninguna falla de guardado muestra un mensaje de éxito ni deja cambios parciales.
- **SC-006**: Una persona modifica un proyecto en tres acciones (Modificar, editar, Guardar), y el formulario y el
  guardado responden en menos de 500 ms con la base local, 50 integrantes en el sistema y 20 Sprints en el proyecto.
- **SC-007**: Cada escenario tiene una prueba que falló antes del código (evidencia RED) y pasa después (evidencia
  GREEN), y la cobertura de `src/project` queda en `cobertura.txt` con al menos 80%.

## Supuestos y decisiones

Decisiones del equipo (no se vuelven a preguntar):

1. Quitar a un integrante con horas se bloquea; "con horas" significa tener registros de horas en historias de este
   proyecto. Las horas en otros proyectos no cuentan.
2. La revisión de Sprints se hace solo si cambian las fechas. El mensaje es "El Sprint N queda fuera de las nuevas
   fechas del proyecto" y nombra el Sprint de menor número afectado.
3. Nombre, fechas e integrantes se guardan en una sola transacción. Guardar sin cambios cuenta como éxito.
4. "Agregar integrante" es marcar integrantes que ya existen; no se crean integrantes nuevos.
5. El texto de éxito es "Proyecto modificado exitosamente". Nombre vacío y último integrante reutilizan los errores
   actuales de HU-01.
6. Última edición gana, pero las reglas de horas y de Sprints se revisan siempre dentro de la transacción de
   guardado, con la fila del proyecto bloqueada.

Supuestos:

- Los mensajes de HU-01 para nombre vacío y para la lista de integrantes son hoy los que muestra HU-21
  ("project name cannot be empty" y "the member list is required"); HU-11 los reutiliza tal cual. Si HU-01 los
  traduce, HU-11 muestra el texto nuevo sin cambios propios.
- La estructura común lee el nombre del proyecto de trabajo en cada solicitud (HU-20b), así que el escenario 3 no
  necesita código propio; se cubre con una prueba.
- La limpieza de la identificación de un integrante quitado (escenario 15) ya la hace HU-20b; HU-11 la cubre con una
  prueba de punta a punta.
- El formulario lista todos los integrantes del sistema, como el alta de HU-21.
- La sección "Proyectos" no exige contexto (HU-21).

## Trazabilidad

| Escenario | Requisitos | Éxito | Prueba de handler / BDD | Prueba de dominio / integración |
|---|---|---|---|---|
| 1 Acceso desde la lista | FR-001, FR-002 | SC-006 | `TestHU11_EnlaceModificarEnLista`, `TestHU11_FormularioConDatosActuales` | `TestPostgresProjectForEdit_ConIntegrantes`, `TestPostgresProjectForEdit_SinIntegrantes` |
| 2a/2b Modificación exitosa | FR-003, FR-004, FR-005 | SC-001 | `TestHU11_ModificacionExitosa`, `TestHU11_SinCambiosEsExito`, BDD "Modificación exitosa" | `TestUpdateProject_DelegaYPropaga`, `TestPostgresUpdate_NombreYFechas`, `TestPostgresUpdate_SinCambios` |
| 3 Nombre en la estructura común | FR-018 | SC-001 | `TestHU11_NombreEnEstructuraComun`, BDD "Nombre visible en la estructura común" | — |
| 4a/4b Nombre vacío | FR-006, FR-017 | SC-002 | `TestHU11_NombreVacio` | `TestUpdateProject_Validaciones` |
| 5 Fechas inválidas | FR-007, FR-017 | SC-002 | `TestHU11_FechasInvalidas` | `TestUpdateProject_Validaciones` |
| 6a/6b/6c Fecha faltante o formato | FR-007, FR-017 | SC-002 | `TestHU11_FechaFaltanteOFormato` | — |
| 7 Agregar integrante | FR-011 | SC-001 | `TestHU11_AgregarIntegrante`, BDD "Agregar integrante" | `TestMembershipChanges`, `TestPostgresUpdate_Alta` |
| 8 Quitar sin horas | FR-010, FR-011 | SC-001 | `TestHU11_QuitarSinHoras` | `TestMembershipChanges`, `TestPostgresUpdate_BajaSinHoras` |
| 9 Quitar el último | FR-008, FR-017 | SC-002 | `TestHU11_QuitarUltimo` | `TestUpdateProject_Validaciones` |
| 10 Quitar con horas | FR-004, FR-010, FR-019, FR-020 | SC-002, SC-003 | `TestHU11_QuitarConHoras`, BDD "Quitar un integrante con horas registradas" | `TestPostgresUpdate_BajaConHoras`, `TestPostgresUpdate_VariosConHoras_NombraAlPrimero`, `TestPostgresUpdate_EsperaCargaDeHoras`, `TestPostgresSaveEntry_RechazaIntegranteQuitado`, `TestPostgresSaveEntry_EsperaBajaEnCurso`, `TestLogEffort_IntegranteQuitadoAlGuardar` |
| 11a Sprints fuera | FR-012, FR-019 | SC-002, SC-004 | `TestHU11_SprintQuedaFuera`, BDD "Fechas que dejan Sprints fuera" | `TestSprintsLeftOutside`, `TestPostgresUpdate_SprintFuera` |
| 11b Sprint que ya estaba fuera | FR-012 | SC-004 | `TestHU11_SprintYaFueraNoBloquea` | `TestSprintsLeftOutside`, `TestPostgresUpdate_SprintYaFuera` |
| 12a/12b Sin cambio de fechas | FR-013 | SC-001 | `TestHU11_SinCambioDeFechasConSprints` | `TestDatesChanged`, `TestPostgresUpdate_SoloNombreConSprints` |
| 13a/13b Integrante inválido | FR-009, FR-017 | SC-002 | `TestHU11_IntegranteInvalido` | `TestUpdateProject_Validaciones`, `TestPostgresUpdate_IntegranteInexistente` |
| 14a/14b Proyecto inexistente | FR-014 | SC-002 | `TestHU11_ProyectoInexistente`, `TestHU11_404AntesQue422`, `TestHU11_ProyectoBorradoAlGuardar` | `TestPostgresUpdate_ProyectoInexistente`, `TestPostgresProjectForEdit_Inexistente` |
| 15 Identificado que fue quitado | FR-011 | — | `TestHU11_IdentificadoQuitado` | — |
| 16 Error al guardar | FR-015, FR-021 | SC-005 | `TestHU11_ErrorAlGuardar`, BDD "Error al guardar" | `TestPostgresUpdate_FalloNoDejaCambios` |
| (HU-20a) Error al leer | FR-016 | — | `TestHU11_ErrorAlLeer` | — |
| (HU-20b) Texto escapado | FR-021 | — | `TestHU11_NombreEscapado` | — |
| (HU-20b) Recarga sin reenviar | FR-003 | SC-001 | `TestHU11_RecargaSinReenviar` | — |
| Concurrencia (última gana) | FR-019 | — | — | `TestPostgresUpdate_ConcurrenteUltimaGana` |
| Rendimiento | — | SC-006 | `TestHU11_Rendimiento` | — |
| Sin efectos fuera de alcance | FR-022, FR-023 | — | — | `TestPostgresUpdate_NombreYFechas` |
