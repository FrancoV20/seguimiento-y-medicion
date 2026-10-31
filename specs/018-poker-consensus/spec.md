# Especificación: HU-18 – Planning Poker: Nueva Ronda y Estimación Acordada

**Rama**: `HU-18`
**Creada**: 2026-10-28
**Estado**: Implementada (pendiente de revisión en el Pull Request)
**Constitución**: v1.1.0
**Depende de**: HU-05b (y por lo tanto HU-05a), HU-02, HU-20a, HU-20b (verificación manual del escenario 10: HU-15 o un
Sprint finalizado cargado a mano)
**Prioridad**: Media (Should Have) · **Valor**: 80 · **Estimación**: 3 SP

**Entrada**: "HU-18 / Planning Poker: Nueva Ronda y Estimación Acordada. Como Scrum Master, quiero iniciar una nueva
ronda cuando hay divergencia y registrar la estimación acordada, para fijar los Story Points de la historia."

## Historia de usuario

Como Scrum Master, quiero iniciar una nueva ronda cuando hay divergencia y registrar la estimación acordada, para
fijar los Story Points de la historia.

> **Alcance**: HU-18 completa el ciclo de Planning Poker de HU-05a y HU-05b. En "Rondas cerradas", la última ronda
> cerrada de cada historia ofrece dos acciones: "Nueva ronda" (abre la ronda siguiente, con votos ocultos como en
> HU-05a) y "Estimación acordada" (fija los Story Points de la historia con un valor de la serie). Las rondas
> anteriores quedan como historial, sin acciones.
>
> **Integrante**: las dos acciones quedan a nombre del integrante identificado en HU-20b. Todavía no hay roles:
> cualquier integrante identificado del proyecto las usa (HU-28 lo restringirá al Scrum Master).
>
> **Sin migración**: la estimación acordada se guarda en los Story Points de la historia; no se guarda qué ronda la
> fijó.

## Clarifications

### Session 2026-10-28

Decisiones del equipo registradas antes de `/speckit-clarify` (no se vuelven a preguntar):

- Q: ¿Cómo se abre una nueva ronda? → A: Con una operación nueva del servicio de Planning Poker y del puerto de
  almacenamiento, con su adaptador PostgreSQL, en una transacción: se busca la ronda del proyecto (sin fila → "La
  ronda no existe"); se bloquea la historia como al abrir la ronda 1 en HU-05a; se leen las rondas de la historia; si
  hay una abierta → "Ya hay una ronda abierta para esta historia"; si la ronda recibida no es la de mayor número → "Hay
  una ronda más reciente de esta historia"; si no, se crea la ronda siguiente (número + 1), abierta, a nombre del
  integrante. El índice de HU-05a que impide dos rondas abiertas por historia es la segunda defensa y su rechazo se
  traduce a "Ya hay una ronda abierta para esta historia". La regla de decisión es una función pura junto a la de
  apertura de HU-05a. (Se amplía en la sesión siguiente: también se rechaza si la historia es de un Sprint
  finalizado.)
- Q: ¿Cómo se registra la estimación acordada? → A: Con otra operación nueva. Primero se valida el valor con la regla
  de HU-05a (vacío → "Seleccioná un valor de la serie"; fuera de la serie → "El valor no pertenece a la serie de
  Fibonacci"; 422, antes que 404/409, como el voto). Después, en una transacción: ronda del proyecto con bloqueo
  exclusivo (sin fila → "La ronda no existe"); abierta → "La ronda sigue abierta: cerrala antes de registrar la
  estimación"; si no es la última de la historia → "Hay una ronda más reciente de esta historia"; Sprint de la
  historia (si tiene) con bloqueo compartido y, si está finalizado → "La historia pertenece a un Sprint finalizado; sus
  Story Points no se pueden modificar"; historia bloqueada; se escriben sus Story Points.
- Q: ¿Hace falta que no haya divergencia? → A: No. Se registra con o sin divergencia. Con divergencia en esa ronda
  (regla de HU-05b), el formulario muestra el recordatorio "La última ronda tuvo divergencia: registrá el valor que
  acordaron después de debatir", con el formato de aviso de HU-20b (texto además del color). Registrar de nuevo
  reemplaza el valor anterior.
- Q: ¿Se guarda qué ronda fijó la estimación? → A: No. Sin migración.
- Q: ¿Cómo se ofrece en la web? → A: Rutas `POST /planning-poker/nueva-ronda` y `POST /planning-poker/estimacion`
  registradas junto a las de HU-05a y HU-05b, con los campos existentes `ronda` y `valor`. En "Rondas cerradas", solo
  la última ronda cerrada de cada historia muestra el botón "Nueva ronda" y el formulario "Estimación acordada" (los
  diez valores de la serie y el botón "Registrar estimación"); las anteriores quedan como historial. Éxitos: "Ronda N
  abierta para la historia «título»" y "Estimación registrada: «título» queda con N Story Points", con redirección 303
  a `/planning-poker`. Rechazos: ronda inexistente 404; ronda abierta de la historia, ronda que no es la última, ronda
  sin cerrar e historia de un Sprint finalizado 409; valor vacío o fuera de la serie 422; otra falla 500 "La operación
  no pudo completarse".
- Q: ¿Dónde se definen textos y errores? → A: Los textos, junto a los de HU-05a/HU-05b en su archivo único de
  mensajes; los errores, junto a los errores del dominio de Planning Poker. Se editan archivos de HU-05a y HU-05b y sus
  pruebas falsas; sus pruebas siguen en verde sin cambiar expectativas.
- Q: ¿Cambia "Abrir ronda" de HU-05a? → A: No. Para una historia que ya tuvo rondas sigue respondiendo "La historia ya
  tuvo una ronda de votación"; las rondas siguientes se abren solo con "Nueva ronda".
- Q: ¿Qué queda afuera? → A: Roles (HU-28), abrir o cerrar rondas automáticamente y guardar la estimación por ronda.

### Session 2026-10-28 (`/speckit-clarify`)

- Q: ¿Se puede abrir una "Nueva ronda" para una historia de un Sprint finalizado (como «Alta»), cuyos Story Points ya
  no se pueden cambiar? → A: No. En la última ronda cerrada de una historia de un Sprint finalizado, "Nueva ronda" y
  "Registrar estimación" aparecen deshabilitados con el motivo "Sprint finalizado" (como "Editar" en HU-13 y las horas
  en HU-17), no ocultos. Si igual se envían, las dos acciones responden 409 "La historia pertenece a un Sprint
  finalizado; sus Story Points no se pueden modificar", comprobado dentro de la transacción (Sprint con bloqueo
  compartido antes que la historia). La lectura de "Rondas cerradas" trae, en la misma consulta, si la historia está en
  un Sprint finalizado. Las historias que volvieron al Backlog al cerrar un Sprint ("Pendiente", sin Sprint) se pueden
  volver a votar y estimar.

## Sección SDD

### Intención

Que la estimación de Planning Poker termine en un dato del Backlog: cuando los votos divergen, el equipo vuelve a
votar después de debatir; cuando acuerda, el Scrum Master fija los Story Points de la historia sin salir de la
pantalla.

### Alcance

- Botón "Nueva ronda" en la última ronda cerrada de cada historia: abre la ronda siguiente, sin votos.
- Formulario "Estimación acordada" en la última ronda cerrada de cada historia: fija los Story Points de la historia
  con un valor de la serie, con o sin divergencia, y con recordatorio si hubo divergencia.
- En las historias de un Sprint finalizado, las dos acciones se muestran deshabilitadas con el motivo "Sprint
  finalizado".
- Rechazos: ronda inexistente o de otro proyecto, ronda abierta de la historia, ronda que no es la última, ronda sin
  cerrar, valor vacío o fuera de la serie, historia de un Sprint finalizado, falla al guardar, sin integrante.

### Fuera de alcance

- Roles y permisos por rol (HU-28).
- Abrir o cerrar rondas automáticamente.
- Guardar la estimación por ronda o un historial de estimaciones (solo queda el valor actual en la historia).
- Cambiar "Abrir ronda" de HU-05a o el cierre de HU-05b.
- Borrar Story Points (la estimación siempre es un valor de la serie).

### Entradas

| Formulario | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `POST /planning-poker/nueva-ronda` | `ronda` | identificador numérico | sí | Última ronda (cerrada) de la historia |
| `POST /planning-poker/estimacion` | `ronda` | identificador numérico | sí | Última ronda (cerrada) de la historia |
| `POST /planning-poker/estimacion` | `valor` | número de la serie | sí | 1, 2, 3, 5, 8, 13, 21, 34, 55 u 89 |

El proyecto y el integrante no son campos: salen del contexto del servidor (HU-20b). Un campo de integrante o de
proyecto enviado en el formulario se ignora.

### Salidas esperadas

- Nueva ronda válida: 303 a `/planning-poker` con "Ronda N abierta para la historia «título»" una sola vez; la ronda N
  aparece entre las abiertas sin votos y las anteriores siguen en "Rondas cerradas".
- Estimación válida: 303 a `/planning-poker` con "Estimación registrada: «título» queda con N Story Points" una sola
  vez; el Backlog muestra la historia con N Story Points.
- Valor vacío o fuera de la serie: 422 con la pantalla y el mensaje; nada cambia.
- Ronda abierta de la historia, ronda que no es la última, ronda sin cerrar o historia de un Sprint finalizado: 409
  con la pantalla y el mensaje; nada cambia.
- Ronda inexistente, de otro proyecto o identificador inválido: 404 con la pantalla y el mensaje; nada cambia.
- Falla al guardar: 500 con la pantalla y "La operación no pudo completarse"; ni rondas nuevas ni Story Points
  cambiados.
- Falla al leer la pantalla: página de error de HU-20a.
- Sin integrante identificado o sin proyecto activo: redirección 303 de HU-20b, con regreso a Planning Poker.

### Reglas de negocio

- **R1 – Última ronda**: la última ronda de una historia es la de mayor número, esté abierta o cerrada. Solo la última
  ronda cerrada de cada historia ofrece acciones.
- **R2 – Nueva ronda**: se abre desde la última ronda de la historia si está cerrada, la historia no tiene una ronda
  abierta y no pertenece a un Sprint finalizado. Su número es el de la última + 1; empieza abierta y sin votos, a nombre de quien la abrió. Si la historia
  tiene una ronda abierta, el rechazo es "Ya hay una ronda abierta para esta historia" aunque la ronda enviada no sea
  la última (escenario 8).
- **R3 – Votos de la nueva ronda**: la ronda nueva funciona igual que la ronda 1 de HU-05a (votos ocultos, cambio de
  voto) y se cierra con HU-05b.
- **R4 – Estimación acordada**: se registra desde la última ronda de la historia, cerrada, con un valor de la serie,
  haya o no divergencia. Reemplaza los Story Points que tuviera la historia. No cambia el estado de la historia ni sus
  rondas.
- **R5 – Orden de las comprobaciones de la estimación**: valor (422) → ronda inexistente (404) → ronda sin cerrar
  (409) → ronda que no es la última (409) → Sprint finalizado (409).
- **R6 – Sprint finalizado**: los Story Points de una historia asignada a un Sprint finalizado no se modifican (misma
  regla que HU-13 y HU-17), así que tampoco se abre una nueva ronda para ella. En su última ronda cerrada, "Nueva ronda"
  y "Registrar estimación" se muestran deshabilitados con el motivo "Sprint finalizado"; si igual se envían, las dos
  responden 409. Una historia sin Sprint (incluida una que volvió al Backlog al cerrar un Sprint) o de un Sprint activo
  se puede volver a votar y estimar.
- **R6b – Orden de las comprobaciones de "Nueva ronda"**: ronda inexistente (404) → ronda abierta en la historia (409)
  → ronda que no es la última (409) → Sprint finalizado (409).
- **R7 – Recordatorio**: si la última ronda cerrada tuvo divergencia (regla de HU-05b), su formulario de estimación
  muestra "La última ronda tuvo divergencia: registrá el valor que acordaron después de debatir".

### Restricciones

- Sin migración y sin dependencias nuevas.
- Solo se editan archivos de HU-05a y HU-05b (dominio de Planning Poker, sus pruebas falsas y la pantalla) y se crean
  archivos nuevos de pruebas; no se edita `src/cmd/server/main.go`.
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript (HU-20a); la protección contra envíos de otro origen es el middleware global de
  HU-20b (los formularios no llevan token).
- Los textos se definen en el archivo único de mensajes de Planning Poker.
- El recordatorio usa el formato de aviso de HU-20b: etiqueta de texto además del color.

### Condiciones de error

| Condición | Mensaje | Respuesta |
|---|---|---|
| Estimación sin valor | "Seleccioná un valor de la serie" | 422 |
| Estimación con un valor fuera de la serie o no numérico | "El valor no pertenece a la serie de Fibonacci" | 422 |
| Ronda inexistente, de otro proyecto o identificador inválido | "La ronda no existe" | 404 |
| Nueva ronda con una ronda abierta en la historia | "Ya hay una ronda abierta para esta historia" | 409 |
| Nueva ronda o estimación desde una ronda que no es la última | "Hay una ronda más reciente de esta historia" | 409 |
| Estimación desde una ronda abierta | "La ronda sigue abierta: cerrala antes de registrar la estimación" | 409 |
| Nueva ronda o estimación de una historia de un Sprint finalizado | "La historia pertenece a un Sprint finalizado; sus Story Points no se pueden modificar" | 409 |
| Falla al guardar (nueva ronda o estimación) | "La operación no pudo completarse" | 500 |
| Falla al leer la pantalla | Página de error de HU-20a | 500 o 503 |
| Sin integrante identificado / sin proyecto activo | Mensaje de HU-20b | 303 |

## Escenarios BDD

Todos los escenarios usan el proyecto «Alfa» con Ana, Beto y Caro y la serie 1, 2, 3, 5, 8, 13, 21, 34, 55, 89 (ver
"Datos de ejemplo").

### Bloque A – Nueva ronda

**Escenario 1: Nueva ronda tras divergencia**
- **Dado** la ronda 1 de «Login» cerrada con divergencia
- **Cuando** presiono "Nueva ronda"
- **Entonces** veo "Ronda 2 abierta para la historia «Login»", la ronda 2 aparece abierta sin votos y la ronda 1 sigue
  en "Rondas cerradas" con Ana 2, Beto 13 y Caro 5

**Escenario 2: Votos ocultos en la nueva ronda**
- **Dado** la ronda 2 de «Login» abierta
- **Cuando** Ana vota 8
- **Entonces** Beto ve que Ana votó, sin ver el valor, igual que en la ronda 1

**Escenario 8: Nueva ronda con una ronda abierta**
- **Dado** «Reportes», que ya tiene la ronda 2 abierta
- **Cuando** se envía otra vez la acción "Nueva ronda" desde su ronda 1
- **Entonces** veo "Ya hay una ronda abierta para esta historia" y no se abre otra

### Bloque B – Estimación acordada

**Escenario 3: Registrar la estimación acordada**
- **Dado** la ronda 2 de «Login» cerrada con Ana 8, Beto 8 y Caro 5 (sin divergencia)
- **Cuando** registro 8 como estimación acordada
- **Entonces** veo "Estimación registrada: «Login» queda con 8 Story Points" y el Backlog muestra «Login» con 8 Story
  Points

**Escenario 4: Consenso directo**
- **Dado** la ronda 1 de «Pagos» cerrada sin divergencia
- **Cuando** registro 5 como estimación acordada
- **Entonces** se registra sin pedir otra ronda y «Pagos» queda con 5 Story Points

**Escenario 5: Registrar después de debatir una divergencia**
- **Dado** la ronda 1 de «Login» cerrada con divergencia y sin una ronda nueva
- **Cuando** abro el formulario de estimación acordada
- **Entonces** veo el recordatorio "La última ronda tuvo divergencia: registrá el valor que acordaron después de
  debatir" y, si registro 5, «Login» queda con 5 Story Points

### Bloque C – Rechazos y errores

**Escenario 6: Valor fuera de la serie**
- **Dado** la ronda 1 de «Pagos» cerrada
- **Cuando** registro 4 como estimación acordada
- **Entonces** veo "El valor no pertenece a la serie de Fibonacci" y «Pagos» no cambia

**Escenario 7: Ronda sin cerrar**
- **Dado** la ronda 2 de «Reportes» abierta
- **Cuando** intento registrar la estimación acordada
- **Entonces** veo "La ronda sigue abierta: cerrala antes de registrar la estimación" y «Reportes» sigue con 3 Story
  Points

**Escenario 9: Ronda que ya no es la última**
- **Dado** «Reportes», con la ronda 2 abierta
- **Cuando** se envía la estimación acordada desde su ronda 1
- **Entonces** veo "Hay una ronda más reciente de esta historia" y «Reportes» no cambia

**Escenario 10: Historia de un Sprint finalizado**
- **Dado** «Alta», del Sprint 1 finalizado, con su ronda 1 cerrada
- **Cuando** registro 5 como estimación acordada
- **Entonces** veo "La historia pertenece a un Sprint finalizado; sus Story Points no se pueden modificar" y «Alta»
  sigue con 3 Story Points

(En la pantalla, la última ronda de «Alta» muestra "Nueva ronda" y "Registrar estimación" deshabilitados con el motivo
"Sprint finalizado"; el escenario envía la estimación igual, como lo haría una pestaña vieja o un envío manual.)

**Escenario 11: Ronda inexistente o de otro proyecto**
- **Dado** una ronda que no existe o que pertenece a otro proyecto
- **Cuando** intento abrir una nueva ronda o registrar la estimación desde ella
- **Entonces** veo "La ronda no existe" y no cambia nada

**Escenario 12: Error al guardar**
- **Dado** un valor válido
- **Cuando** ocurre un error al guardar
- **Entonces** veo "La operación no pudo completarse" y los Story Points no cambian

### Datos de ejemplo

Proyecto «Alfa», integrantes Ana, Beto y Caro. Niveles de la serie (HU-05b): 1→1, 2→2, 3→3, 5→4, 8→5, 13→6, 21→7,
34→8, 55→9, 89→10; hay divergencia si la diferencia de niveles entre el menor y el mayor voto es mayor que 2.

| Historia | Story Points al empezar | Sprint | Rondas al empezar | Rango (niveles) | Divergencia |
|---|---|---|---|---|---|
| «Login» | sin Story Points | ninguno | Ronda 1 cerrada: Ana 2, Beto 13, Caro 5 | 2 a 13 (2 y 6: 4) | Sí |
| «Pagos» | sin Story Points | ninguno | Ronda 1 cerrada: Ana 3, Beto 5, Caro 8 | 3 a 8 (3 y 5: 2) | No |
| «Reportes» | 3 | ninguno | Ronda 1 cerrada: Ana 3, Beto 13, Caro 3; ronda 2 abierta con el voto de Ana | 3 a 13 (3 y 6: 3) | Sí (ronda 1) |
| «Alta» | 3 | Sprint 1, finalizado | Ronda 1 cerrada: Ana 5, Beto 5, Caro 5 | 5 a 5 (4 y 4: 0) | No |

Cuentas:

- Escenario 1: «Login» tiene solo la ronda 1 → "Nueva ronda" crea la ronda 1 + 1 = 2.
- Escenario 3: ronda 2 de «Login» con 8, 8 y 5 → de 5 a 8, niveles 4 y 5: 1 ≤ 2 → sin divergencia (coincide con el
  Issue). 8 es de la serie → «Login» pasa de sin Story Points a 8.
- Escenario 4: «Pagos» 3, 5, 8 → diferencia 2, el límite exacto sin divergencia; 5 es de la serie → de sin Story Points
  a 5.
- Escenario 5: «Login» 2, 13, 5 → diferencia 4 > 2 → recordatorio; 5 → «Login» queda con 5.
- Escenario 6: 4 no está en la serie → 422 antes de mirar la ronda; «Pagos» sigue sin Story Points.
- Escenarios 7, 8 y 9: «Reportes» tiene las rondas 1 (cerrada) y 2 (abierta); la última es la 2. Estimar desde la 2 →
  sigue abierta; "Nueva ronda" desde la 1 → hay una abierta (gana sobre "no es la última"); estimar desde la 1 → está
  cerrada pero no es la última. «Reportes» sigue con 3.
- Escenario 10: «Alta» 5, 5, 5 → sin divergencia; su ronda 1 es la última y está cerrada, pero el Sprint 1 está
  finalizado → acciones deshabilitadas y 409 si se envían; «Alta» sigue con 3 y no se crea la ronda 2.

### Casos límite

- Un `ronda` vacío, no numérico, negativo, con ceros a la izquierda o mayor que el máximo se trata como inexistente
  (404) sin consultar la base; en la estimación, el valor se valida antes (un valor vacío con una ronda inexistente
  responde 422).
- Una ronda de otro proyecto se trata como inexistente (404) en las dos acciones, aunque esté cerrada o sea la última.
- "Nueva ronda" desde una ronda cerrada que no es la última, con todas las rondas de la historia cerradas → 409 "Hay
  una ronda más reciente de esta historia".
- "Nueva ronda" desde la ronda abierta de la historia → 409 "Ya hay una ronda abierta para esta historia".
- Dos "Nueva ronda" simultáneas desde la misma ronda: una abre la ronda siguiente y la otra recibe 409 "Ya hay una
  ronda abierta para esta historia"; queda una sola ronda nueva.
- "Nueva ronda" y "Estimación acordada" simultáneas desde la misma ronda: una espera a la otra; si la nueva ronda llega
  primero, la estimación recibe "Hay una ronda más reciente de esta historia".
- Una estimación mientras se cierra el Sprint de la historia: si el cierre llega primero, la estimación espera y recibe
  "La historia pertenece a un Sprint finalizado…"; si la estimación llega primero, el cierre espera y toma los Story
  Points nuevos. Nunca se bloquean entre sí.
- Registrar de nuevo desde la misma ronda reemplaza el valor anterior (por ejemplo, «Pagos» de 5 a 8).
- Registrar el mismo valor que ya tiene la historia es un éxito y la deja igual.
- Una historia de un Sprint activo se puede volver a votar y estimar. Una historia que volvió al Backlog al cerrar un
  Sprint ("Pendiente", sin Sprint) también. Una historia de un Sprint finalizado no admite ninguna de las dos acciones.
- "Nueva ronda" desde la última ronda de «Alta» (Sprint finalizado) → 409 "La historia pertenece a un Sprint
  finalizado…" y no se crea la ronda 2.
- La ronda nueva de una historia aparece entre las abiertas con su número ("Ronda 2"); sus votos se ocultan como en
  HU-05a y se cierran con HU-05b.
- Las rondas cerradas anteriores de una historia siguen en "Rondas cerradas" con sus votos, sin acciones.
- Recargar la página después de cualquiera de las dos acciones no reenvía el formulario ni repite el mensaje.
- Los títulos de las historias se muestran escapados, también en los mensajes de éxito.
- "Abrir ronda" (HU-05a) sobre una historia con rondas sigue respondiendo "La historia ya tuvo una ronda de votación".

## Requisitos funcionales

- **FR-001**: En "Rondas cerradas", la última ronda cerrada de cada historia (R1) MUST ofrecer el botón "Nueva ronda"
  (`POST /planning-poker/nueva-ronda`) y el formulario "Estimación acordada" (`POST /planning-poker/estimacion`) con
  los diez valores de la serie; las demás rondas cerradas MUST NOT ofrecer acciones. Si la historia pertenece a un
  Sprint finalizado, las dos acciones MUST mostrarse deshabilitadas con el motivo "Sprint finalizado"; ese dato MUST
  leerse en la misma lectura que las rondas cerradas.
- **FR-002**: "Nueva ronda" desde la última ronda cerrada de una historia sin rondas abiertas MUST crear la ronda
  siguiente (R2) a nombre del integrante identificado, responder 303 a `/planning-poker` y mostrar una sola vez "Ronda N
  abierta para la historia «título»".
- **FR-003**: Si la historia tiene una ronda abierta, "Nueva ronda" MUST responder 409 con "Ya hay una ronda abierta
  para esta historia" y MUST NOT crear rondas, aunque la ronda enviada no sea la última.
- **FR-004**: Si la ronda enviada no es la última de su historia (y no hay rondas abiertas, para "Nueva ronda"), las
  dos acciones MUST responder 409 con "Hay una ronda más reciente de esta historia" y MUST NOT modificar nada.
- **FR-005**: La ronda nueva MUST comportarse como una ronda de HU-05a (votos ocultos, cambio de voto) y cerrarse con
  HU-05b; las rondas cerradas anteriores MUST seguir mostrándose con sus votos.
- **FR-006**: La estimación acordada con un valor de la serie desde la última ronda cerrada MUST escribir ese valor en
  los Story Points de la historia (reemplazando el anterior), responder 303 a `/planning-poker` y mostrar una sola vez
  "Estimación registrada: «título» queda con N Story Points".
- **FR-007**: La estimación MUST permitirse con o sin divergencia; si la ronda tuvo divergencia, el formulario MUST
  mostrar el recordatorio de R7 con el formato de aviso de HU-20b.
- **FR-008**: Una estimación sin valor MUST responder 422 con "Seleccioná un valor de la serie"; un valor fuera de la
  serie o no numérico MUST responder 422 con "El valor no pertenece a la serie de Fibonacci". El valor MUST validarse
  antes que la ronda y MUST NOT modificarse nada.
- **FR-009**: Una estimación desde una ronda abierta MUST responder 409 con "La ronda sigue abierta: cerrala antes de
  registrar la estimación" y MUST NOT modificar la historia.
- **FR-010**: "Nueva ronda" y la estimación de una historia asignada a un Sprint finalizado MUST responder 409 con "La
  historia pertenece a un Sprint finalizado; sus Story Points no se pueden modificar", comprobado dentro de la
  transacción, y MUST NOT crear rondas ni modificar la historia.
- **FR-011**: Si la ronda está ausente, no es un identificador válido, no existe o es de otro proyecto, las dos
  acciones MUST responder 404 con "La ronda no existe" y MUST NOT modificar nada.
- **FR-012**: Si falla el guardado de cualquiera de las dos acciones, el sistema MUST responder 500 con "La operación
  no pudo completarse", MUST NOT mostrar un mensaje de éxito y MUST NOT dejar rondas nuevas ni Story Points cambiados.
- **FR-013**: Dos "Nueva ronda" simultáneas sobre la misma historia MUST terminar con una sola ronda nueva; una
  estimación y un cierre de Sprint simultáneos MUST serializarse sin bloquearse entre sí (orden Sprint → historia).
- **FR-014**: Sin integrante identificado o sin proyecto activo MUST aplicarse la redirección de HU-20b, con regreso a
  Planning Poker, y MUST NOT crearse rondas ni cambiarse Story Points.
- **FR-015**: El proyecto y el integrante MUST salir del contexto del servidor; todas las lecturas y escrituras MUST
  filtrarse por el proyecto activo.
- **FR-016**: Si falla la lectura de la pantalla, el sistema MUST mostrar la página de error de HU-20a.
- **FR-017**: Los títulos de las historias MUST mostrarse escapados, también en los mensajes de éxito.
- **FR-018**: HU-18 MUST NOT agregar migraciones, guardar la estimación por ronda, cambiar el estado de la historia ni
  cambiar el comportamiento de "Abrir ronda", votar y cerrar (HU-05a y HU-05b).
- **FR-019**: Los textos nuevos MUST definirse junto a los de HU-05a y HU-05b; las respuestas de error MUST NOT incluir
  texto de consultas ni detalles internos.

## Entidades clave

- **Ronda de votación** (HU-05a, HU-05b): historia, número, estado, quién la abrió, fechas de apertura y cierre. HU-18
  crea rondas con número mayor que 1.
- **Historia de usuario** (HU-02): HU-18 escribe sus Story Points; no cambia nada más.
- **Sprint** (HU-04, HU-15): activo o finalizado; una historia asignada a un Sprint finalizado no cambia sus Story
  Points.
- **Rango de la ronda** (HU-05b, no se guarda): menor, mayor y si hay divergencia; decide el recordatorio.
- **Integrante del proyecto** (HU-01, HU-20b): quien abre la nueva ronda y registra la estimación.

## Criterios de éxito

- **SC-001**: En el 100% de las pruebas, el número de cada ronda nueva y los Story Points resultantes coinciden con las
  cuentas a mano de "Datos de ejemplo".
- **SC-002**: Ningún envío rechazado (valor inválido, ronda inexistente, abierta, que no es la última, Sprint
  finalizado, sin integrante o con falla) crea rondas ni cambia Story Points.
- **SC-003**: Dos "Nueva ronda" simultáneas dejan exactamente una ronda nueva; una estimación concurrente con el cierre
  del Sprint termina en éxito o en 409, nunca en un bloqueo mutuo ni en Story Points cambiados en un Sprint finalizado.
- **SC-004**: En la ronda nueva, el valor de un voto ajeno no aparece en ninguna parte de la respuesta (como en
  HU-05a).
- **SC-005**: El Scrum Master fija los Story Points con dos acciones (elegir el valor y "Registrar estimación") sin
  salir de Planning Poker, y la pantalla responde en menos de 500 ms con la base local, 20 rondas abiertas, 50 rondas
  cerradas y 10 integrantes.
- **SC-006**: Cada bloque tiene pruebas que fallaron antes del código (evidencia RED) y pasan después (evidencia
  GREEN), y la cobertura de `src/poker` queda en `cobertura.txt` con al menos 80%.

## Supuestos y decisiones

- No hay roles: cualquier integrante identificado abre rondas nuevas y registra la estimación (HU-28 lo restringirá).
- La identificación de HU-20b valida en cada solicitud que el integrante pertenezca al proyecto activo; HU-18 no
  repite esa comprobación.
- El Sprint de una historia es el de su asignación (una historia tiene como máximo una); HU-15 conserva la asignación
  de las historias completadas al finalizar el Sprint. HU-13, HU-15 y HU-17 todavía no están en el código: HU-18 aplica
  la misma regla por su cuenta.
- El Issue no da los votos de la ronda 1 de «Reportes» ni el valor del voto de Ana en su ronda 2: se usan los de
  HU-05b para la ronda 1 (Ana 3, Beto 13, Caro 3, con divergencia de 3 a 13) y 5 para el voto de Ana en la ronda 2.
- El Issue solo menciona un Sprint para «Alta»: «Login», «Pagos» y «Reportes» se toman sin Sprint (no cambia ninguna
  cuenta; un Sprint activo no impediría estimarlas).
- El Issue no da las fechas ni el objetivo del Sprint 1 de «Alta»: las pruebas lo cargan directo en la base como
  finalizado, con «Alta» asignada y en "Hecho", como lo deja HU-15.
- "Nueva ronda" no exige divergencia: está disponible en toda última ronda cerrada (notas del equipo).
- Los textos son los del Issue y las notas; se definen en el archivo único de mensajes.

## Trazabilidad

| Escenario | Requisitos | Éxito | Prueba de handler / BDD | Prueba de dominio / integración |
|---|---|---|---|---|
| 1 Nueva ronda tras divergencia | FR-001, FR-002, FR-005 | SC-001 | `TestHU18_AccionesSoloEnLaUltimaRonda`, `TestHU18_NuevaRonda`, BDD `TestBDD_HU18_NuevaRondaTrasDivergencia` | `TestCheckNextRound`, `TestService_OpenNextRound_OK`, `TestPostgresOpenNextRound_OK`, `TestPostgresPokerCatalog_UltimaRonda` |
| 2 Votos ocultos en la nueva ronda | FR-005 | SC-004 | `TestHU18_VotosOcultosEnLaNuevaRonda`, BDD `TestBDD_HU18_VotosOcultosEnLaNuevaRonda` | — (HU-05a ya prueba el ocultamiento por construcción) |
| 3 Registrar la estimación acordada | FR-006 | SC-001 | `TestHU18_RegistrarEstimacion`, BDD `TestBDD_HU18_RegistrarEstimacionAcordada` | `TestService_AgreeEstimate_OK`, `TestPostgresAgreeEstimate_OK` |
| 4 Consenso directo | FR-006, FR-007 | SC-001 | `TestHU18_ConsensoDirecto`, BDD `TestBDD_HU18_ConsensoDirecto` | `TestCheckAgree`, `TestPostgresAgreeEstimate_OK` |
| 5 Registrar después de debatir | FR-006, FR-007 | SC-001 | `TestHU18_RecordatorioDeDivergencia`, BDD `TestBDD_HU18_RegistrarDespuesDeDebatir` | `TestPostgresAgreeEstimate_ConDivergencia` |
| 6 Valor fuera de la serie | FR-008 | SC-002 | `TestHU18_ValorFueraDeSerie`, BDD `TestBDD_HU18_ValorFueraDeLaSerie` | `TestService_AgreeEstimate_Validaciones` |
| 7 Ronda sin cerrar | FR-009 | SC-002 | `TestHU18_RondaSinCerrar`, BDD `TestBDD_HU18_RondaSinCerrar` | `TestCheckAgree`, `TestPostgresAgreeEstimate_RondaAbierta` |
| 8 Nueva ronda con una ronda abierta | FR-003 | SC-002 | `TestHU18_NuevaRondaConRondaAbierta`, BDD `TestBDD_HU18_NuevaRondaConRondaAbierta` | `TestCheckNextRound`, `TestPostgresOpenNextRound_ConRondaAbierta` |
| 9 Ronda que ya no es la última | FR-004 | SC-002 | `TestHU18_RondaQueNoEsLaUltima`, BDD `TestBDD_HU18_RondaQueYaNoEsLaUltima` | `TestCheckAgree`, `TestPostgresAgreeEstimate_NoEsLaUltima`, `TestPostgresOpenNextRound_NoEsLaUltima` |
| 10 Historia de un Sprint finalizado | FR-001, FR-010 | SC-002 | `TestHU18_SprintFinalizado`, BDD `TestBDD_HU18_HistoriaDeUnSprintFinalizado` | `TestCheckNextRound`, `TestCheckAgree`, `TestPostgresAgreeEstimate_SprintFinalizado`, `TestPostgresOpenNextRound_SprintFinalizado`, `TestPostgresPokerCatalog_SprintFinalizado` |
| 11 Ronda inexistente o de otro proyecto | FR-011, FR-015 | SC-002 | `TestHU18_RondaInexistente`, BDD `TestBDD_HU18_RondaInexistente` | `TestService_OpenNextRound_Validaciones`, `TestService_AgreeEstimate_Validaciones`, `TestPostgresOpenNextRound_Inexistente`, `TestPostgresOpenNextRound_OtroProyecto`, `TestPostgresAgreeEstimate_Inexistente`, `TestPostgresAgreeEstimate_OtroProyecto` |
| 12 Error al guardar | FR-012, FR-019 | SC-002 | `TestHU18_ErrorAlGuardar` | `TestPostgresOpenNextRound_FalloNoDejaCambios`, `TestPostgresAgreeEstimate_FalloNoDejaCambios` |
| (Caso límite) Dos "Nueva ronda" a la vez | FR-013 | SC-003 | — | `TestPostgresOpenNextRound_Concurrente` |
| (Caso límite) Estimación y cierre de Sprint | FR-010, FR-013 | SC-003 | — | `TestPostgresAgreeEstimate_CierreDeSprintPrimero`, `TestPostgresAgreeEstimate_EstimacionPrimero` |
| (Caso límite) "Nueva ronda" y estimación desde la misma ronda | FR-004, FR-013 | SC-003 | — | `TestPostgresNextRoundYEstimacion_Serializadas` |
| (Caso límite) Historia de un Sprint activo | FR-006, FR-010 | — | — | `TestPostgresAgreeEstimate_SprintActivo` |
| (HU-20b) Sin integrante identificado | FR-014 | SC-002 | `TestHU18_SinIdentificarme` | — |
| (HU-20a) Error al leer | FR-016 | — | `TestHU05a_ErrorAlLeer` (sigue en verde) | — |
| (HU-20b) Texto escapado | FR-017 | — | `TestHU18_TituloEscapado` | — |
| (HU-20b) Recarga sin reenviar | FR-002, FR-006 | — | `TestHU18_RecargaSinReenviar` | — |
| Rendimiento de la pantalla | — | SC-005 | `TestHU18_Rendimiento` | — |
| Sin migración ni cambios en HU-05a/HU-05b | FR-018 | SC-006 | Pruebas de HU-05a y HU-05b en cada GREEN | `TestPostgresAgreeEstimate_OK` (solo cambia `story_points`), revisión en T027 |
