# Especificación: HU-05b – Revelación de Votos y Detección de Divergencia

**Rama**: `HU-05b`
**Creada**: 2026-10-28
**Estado**: Implementada (pendiente de revisión en el Pull Request)
**Constitución**: v1.1.0
**Depende de**: HU-05a, HU-20a, HU-20b
**Habilita**: HU-18 (nueva ronda y estimación acordada), que reutiliza la regla de divergencia
**Prioridad**: Media-Alta (Should Have) · **Valor**: 85 · **Estimación**: 3 SP

**Entrada**: "HU-05b – Revelación de Votos y Detección de Divergencia. Como Scrum Master, quiero cerrar la ronda de
votación y ver los votos revelados con las divergencias marcadas, para poder guiar la discusión del equipo hacia un
consenso."

## Historia de usuario

Como Scrum Master, quiero cerrar la ronda de votación y ver los votos revelados con las divergencias marcadas, para
poder guiar la discusión del equipo hacia un consenso.

> **Alcance**: HU-05b agrega a "Planning Poker" (HU-05a) el botón "Cerrar ronda" en cada ronda abierta y una sección
> "Rondas cerradas" con los votos revelados, el rango de los votos y la alerta de divergencia. Cerrar una ronda es
> definitivo: después nadie vota en ella. Abrir una ronda nueva para la misma historia y registrar la estimación
> acordada son de HU-18.
>
> **Integrante**: el cierre lo hace el integrante identificado en HU-20b. Todavía no hay roles: cualquier integrante
> identificado del proyecto cierra rondas (HU-28 lo restringirá al Scrum Master).
>
> **Sin migración**: las tablas de HU-05a ya tienen el estado y la fecha de cierre de la ronda; la divergencia se
> calcula al mostrar, no se guarda.

## Clarifications

### Session 2026-10-28

Decisiones del equipo registradas antes de `/speckit-clarify` (no se vuelven a preguntar):

- Q: ¿Dónde vive la regla de divergencia? → A: En el dominio de Planning Poker, como regla pura en un archivo nuevo:
  recibe los valores emitidos y devuelve el menor, el mayor y si hay divergencia. Nivel = posición del valor en la
  serie (1 → 1, 2 → 2, 3 → 3, 5 → 4, 8 → 5, 13 → 6, 21 → 7, 34 → 8, 55 → 9, 89 → 10); hay divergencia si
  nivel(mayor) − nivel(menor) > 2. Un solo voto nunca diverge. Es el contrato que usa HU-18 (nombre y firma en
  `data-model.md`).
- Q: ¿Cómo se cierra una ronda? → A: Con una operación nueva del servicio de Planning Poker y del puerto de
  almacenamiento, con su adaptador PostgreSQL, en una sola transacción: se toma la ronda del proyecto con bloqueo
  exclusivo (sin fila → "La ronda no existe"; ya cerrada → "La ronda ya está cerrada"), se cuentan sus votos (0 →
  "La ronda no tiene votos") y se cambia el estado a "Cerrada" con la fecha de cierre en una sola actualización. Sin
  migración.
- Q: ¿Hace falta que voten todos? → A: No. Se cierra con al menos un voto; los integrantes del proyecto que no votaron
  aparecen como "No votó". Lo cierra cualquier integrante identificado (la sección ya lo exige; sin roles hasta
  HU-28).
- Q: ¿Cómo se ofrece en la web? → A: Ruta `POST /planning-poker/cerrar` registrada junto a las de HU-05a, campo
  `ronda`, botón "Cerrar ronda" en cada ronda abierta. Éxito: mensaje "Ronda cerrada para la historia «título»" y
  redirección 303 a `/planning-poker`. Rechazos: ronda inexistente o de otro proyecto → 404 "La ronda no existe";
  ronda ya cerrada → 409 "La ronda ya está cerrada"; ronda sin votos → 409 "La ronda no tiene votos"; cualquier otra
  falla → 500 "La operación no pudo completarse". El 404 se decide antes que el 409.
- Q: ¿Cómo se muestran las rondas cerradas? → A: Sección nueva "Rondas cerradas", de la más recientemente cerrada a
  la más antigua (empate: la de identificador mayor primero). Por ronda: historia, número de ronda, cada integrante
  del proyecto con su valor o "No votó" (ordenados por nombre) y, debajo, el rango: con divergencia, la alerta
  "Divergencia: los votos van de X a Y. Debatan antes de acordar la estimación" con el formato de alerta de HU-20b
  (etiqueta de texto además del color); sin divergencia, "Sin divergencia: los votos van de X a Y". Sin rondas
  cerradas: "Aún no hay rondas cerradas". Los valores se leen siempre y solo de rondas cerradas; una prueba verifica
  que el voto de una ronda abierta nunca aparece. (La lista se amplía en la sesión siguiente: también figura quien
  votó y ya no pertenece al proyecto.)
- Q: ¿Dónde se definen los textos y errores nuevos? → A: Los textos, junto a los de HU-05a en su archivo único de
  mensajes; los errores, junto a los errores del dominio de HU-05a.
- Q: ¿Qué archivos de HU-05a se editan? → A: Los del dominio (puerto, servicio, adaptador, errores) y sus pruebas
  falsas, y en la web los handlers, el formulario, los mensajes, el catálogo, su adaptador y la plantilla. No cambia
  el comportamiento de abrir ni de votar: las pruebas de HU-05a siguen en verde.
- Q: ¿Qué queda afuera? → A: Abrir una nueva ronda y registrar la estimación acordada (HU-18), roles (HU-28) y
  cerrar la ronda automáticamente cuando votan todos.

### Session 2026-10-28 (duda de `/speckit-specify`, revisada en `/speckit-clarify`)

- Q: Si un integrante votó y después lo quitaron del proyecto, ¿su voto se muestra y cuenta en la ronda cerrada? → A:
  Se muestra y cuenta. La lista revelada son los integrantes actuales más quien votó aunque ya no pertenezca; el rango
  y la divergencia usan todos los votos guardados de la ronda. Así una ronda cerrada por la aplicación nunca queda sin
  valores visibles y HU-18 recibe los mismos valores que se ven.

## Sección SDD

### Intención

Que el equipo vea, al mismo tiempo y todos igual, qué estimó cada integrante, y que la pantalla marque cuándo los
votos están tan separados que conviene discutir antes de acordar. Así el Scrum Master guía la conversación hacia
quienes votaron en los extremos.

### Alcance

- Botón "Cerrar ronda" en cada ronda abierta de "Planning Poker".
- Cerrar una ronda abierta con al menos un voto.
- Sección "Rondas cerradas" con historia, número de ronda, votos revelados (valor o "No votó") y rango con o sin
  alerta de divergencia, igual para todos los integrantes.
- Regla pura de divergencia reutilizable por HU-18.
- Rechazos: ronda sin votos, ya cerrada, inexistente o de otro proyecto, sin integrante identificado, falla al
  guardar.

### Fuera de alcance

- Abrir una nueva ronda para la misma historia y registrar la estimación acordada (HU-18): HU-05b no modifica la
  historia ni sus Story Points.
- Roles y permisos por rol (HU-28): el Scrum Master todavía no se distingue de los demás integrantes.
- Cerrar la ronda automáticamente cuando votan todos.
- Reabrir una ronda cerrada o borrar votos.
- Guardar el resultado de la divergencia (se calcula al mostrar).
- Mostrar quién cerró la ronda (no se guarda; sin migración).

### Entradas

| Formulario | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `POST /planning-poker/cerrar` | `ronda` | identificador numérico | sí | Ronda del proyecto activo |

El proyecto y el integrante no son campos: salen del contexto del servidor (HU-20b). Un campo de integrante o de
proyecto enviado en el formulario se ignora.

### Salidas esperadas

- Cierre válido: redirección 303 a `/planning-poker` con "Ronda cerrada para la historia «título»" una sola vez; la
  ronda deja de figurar entre las abiertas y aparece primera en "Rondas cerradas".
- Ronda sin votos o ya cerrada: 409 con la pantalla y el mensaje; nada cambia.
- Ronda inexistente, de otro proyecto o identificador inválido: 404 con la pantalla y el mensaje; nada cambia.
- Falla al guardar el cierre: 500 con la pantalla y "La operación no pudo completarse"; la ronda sigue abierta y sus
  votos ocultos.
- Falla al leer la pantalla: página de error de HU-20a.
- Sin integrante identificado o sin proyecto activo: redirección 303 de HU-20b, con regreso a Planning Poker.

### Reglas de negocio

- **R1 – Nivel**: el nivel de un valor es su posición en la serie 1, 2, 3, 5, 8, 13, 21, 34, 55, 89 (1 es el nivel 1,
  89 el nivel 10).
- **R2 – Divergencia**: con los valores emitidos en una ronda, hay divergencia si nivel(mayor) − nivel(menor) > 2.
  Con un solo voto, o con todos los votos iguales, no hay divergencia.
- **R3 – Rango**: el rango va del menor al mayor valor emitido ("de X a Y"; con un solo valor, "de X a X"). Solo
  cuentan los votos emitidos: "No votó" no es un valor.
- **R4 – Cierre**: solo se cierra una ronda abierta del proyecto activo con al menos un voto. El cierre es definitivo
  y registra la fecha de cierre.
- **R5 – Revelación**: los valores de una ronda se muestran solo cuando está cerrada, y a todos los integrantes por
  igual. Mientras está abierta rige el ocultamiento de HU-05a.
- **R6 – Votos después del cierre**: un voto en una ronda cerrada se rechaza (HU-05a); un voto que llega mientras se
  cierra queda registrado antes del cierre o se rechaza, nunca después.
- **R7 – Lista revelada**: cada ronda cerrada muestra a los integrantes actuales del proyecto y, además, a quien votó
  en ella aunque ya no pertenezca al proyecto, ordenados por nombre, con su valor o "No votó". El rango usa todos los
  votos de la ronda, así que lo que se ve es exactamente lo que se calcula.

### Restricciones

- Sin migración y sin dependencias nuevas.
- Solo se editan archivos de HU-05a (dominio de Planning Poker, sus pruebas falsas y la pantalla) y se crean archivos
  nuevos de código y de pruebas; no se edita `src/cmd/server/main.go` porque la ruta nueva se registra con la sección
  existente.
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript (HU-20a); la protección contra envíos de otro origen es el middleware global de
  HU-20b (los formularios no llevan token).
- Los textos se definen en el archivo único de mensajes de Planning Poker.
- La alerta de divergencia usa el formato de alerta de HU-20b: etiqueta de texto además del color.

### Condiciones de error

| Condición | Mensaje | Respuesta |
|---|---|---|
| Cerrar una ronda sin votos | "La ronda no tiene votos" | 409 |
| Cerrar una ronda ya cerrada | "La ronda ya está cerrada" | 409 |
| Votar en una ronda cerrada (HU-05a) | "La ronda ya está cerrada" | 409 |
| Ronda inexistente, de otro proyecto o identificador inválido | "La ronda no existe" | 404 |
| Falla al guardar el cierre | "La operación no pudo completarse" | 500 |
| Falla al leer la pantalla | Página de error de HU-20a | 500 o 503 |
| Sin integrante identificado / sin proyecto activo | Mensaje de HU-20b | 303 |

## Escenarios BDD

Todos los escenarios usan el proyecto «Alfa» con Ana, Beto y Caro y la serie 1, 2, 3, 5, 8, 13, 21, 34, 55, 89 (ver
"Datos de ejemplo").

### Bloque A – Cerrar y revelar

**Escenario 1: Revelación de votos**
- **Dado** la ronda abierta de «Login», en la que votaron Ana, Beto y Caro
- **Cuando** cierro la ronda
- **Entonces** veo "Ronda cerrada para la historia «Login»" y los votos revelados: Ana 2, Beto 13 y Caro 5

**Escenario 5: Cerrar sin que todos hayan votado**
- **Dado** la ronda abierta de «Perfil», en la que Caro no votó
- **Cuando** cierro la ronda
- **Entonces** la ronda se cierra, veo Ana 5, Beto 5 y Caro "No votó", y veo "Sin divergencia: los votos van de 5 a 5"

**Escenario 7: Los votos se ven para todos**
- **Dado** la ronda de «Login» cerrada por Ana
- **Cuando** Beto abre "Planning Poker"
- **Entonces** ve los mismos votos revelados y la misma alerta

### Bloque B – Divergencia

**Escenario 2: Alerta de divergencia**
- **Dado** la ronda de «Login» cerrada con votos de 2, 13 y 5
- **Cuando** veo sus resultados
- **Entonces** veo la alerta "Divergencia: los votos van de 2 a 13. Debatan antes de acordar la estimación"

**Escenario 3: Sin divergencia en el límite**
- **Dado** la ronda de «Pagos» cerrada con votos de 3, 5 y 8 (dos niveles entre el menor y el mayor)
- **Cuando** veo sus resultados
- **Entonces** veo "Sin divergencia: los votos van de 3 a 8" y no veo la alerta

**Escenario 4: Divergencia de tres niveles**
- **Dado** la ronda de «Reportes» cerrada con votos de 3, 13 y 3
- **Cuando** veo sus resultados
- **Entonces** veo la alerta "Divergencia: los votos van de 3 a 13. Debatan antes de acordar la estimación"

### Bloque C – Rechazos y errores

**Escenario 6: Ronda sin votos**
- **Dado** la ronda abierta de «Ayuda», sin votos
- **Cuando** intento cerrarla
- **Entonces** veo "La ronda no tiene votos" y la ronda sigue abierta

**Escenario 8: Votar en una ronda cerrada**
- **Dado** la ronda de «Login» cerrada
- **Cuando** Caro intenta votar 8
- **Entonces** ve "La ronda ya está cerrada" y su voto sigue siendo 5

**Escenario 9: Cerrar dos veces**
- **Dado** la ronda de «Login» cerrada
- **Cuando** se envía otra vez la acción de cerrarla
- **Entonces** veo "La ronda ya está cerrada" y no cambia nada

**Escenario 10: Ronda inexistente o de otro proyecto**
- **Dado** una ronda que no existe o que pertenece a otro proyecto
- **Cuando** intento cerrarla
- **Entonces** veo "La ronda no existe" y no cambia nada

**Escenario 11: Sin integrante identificado**
- **Dado** que no me identifiqué como integrante
- **Cuando** intento cerrar una ronda
- **Entonces** se me pide identificarme y no se cierra nada

**Escenario 12: Error al cerrar**
- **Dado** una ronda con votos
- **Cuando** ocurre un error al guardar el cierre
- **Entonces** veo "La operación no pudo completarse", la ronda sigue abierta y sus votos siguen ocultos

### Datos de ejemplo

Proyecto «Alfa», integrantes Ana, Beto y Caro. Serie y niveles (R1):

| Valor | 1 | 2 | 3 | 5 | 8 | 13 | 21 | 34 | 55 | 89 |
|---|---|---|---|---|---|---|---|---|---|---|
| Nivel | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 |

| Ronda (abierta al empezar) | Ana | Beto | Caro | Menor (nivel) | Mayor (nivel) | Diferencia | Resultado esperado |
|---|---|---|---|---|---|---|---|
| Ronda 1 de «Login» | 2 | 13 | 5 | 2 (2) | 13 (6) | 4 | Divergencia: los votos van de 2 a 13 |
| Ronda 1 de «Pagos» | 3 | 5 | 8 | 3 (3) | 8 (5) | 2 | Sin divergencia: los votos van de 3 a 8 |
| Ronda 1 de «Reportes» | 3 | 13 | 3 | 3 (3) | 13 (6) | 3 | Divergencia: los votos van de 3 a 13 |
| Ronda 1 de «Perfil» | 5 | 5 | No votó | 5 (4) | 5 (4) | 0 | Sin divergencia: los votos van de 5 a 5 |
| Ronda 1 de «Ayuda» | — | — | — | — | — | — | No se cierra: "La ronda no tiene votos" |

Cuentas:

- «Login»: 6 − 2 = 4 > 2 → divergencia (coincide con "2 y 13 sí (4)" de las notas del equipo).
- «Pagos»: 5 − 3 = 2, que no es mayor que 2 → sin divergencia; es el límite exacto (escenario 3).
- «Reportes»: 6 − 3 = 3 > 2 → divergencia; el 3 repetido no cambia el menor (escenario 4).
- «Perfil»: dos votos de 5 → 4 − 4 = 0 → sin divergencia; Caro figura como "No votó" y no entra en el rango.
- «Ayuda»: 0 votos → el cierre se rechaza y la ronda sigue abierta.
- Escenarios 8 y 9: después de cerrar «Login», Caro sigue con 5 y el rango sigue "de 2 a 13".

### Casos límite

- Un `ronda` vacío, no numérico, negativo, con ceros a la izquierda o mayor que el máximo se trata como inexistente
  (404) sin consultar la base.
- Una ronda de otro proyecto se trata como inexistente (404) aunque esté cerrada o sin votos: el 404 se decide antes
  que el 409 y no revela datos de otros proyectos.
- Una ronda ya cerrada se rechaza con "La ronda ya está cerrada" antes de mirar sus votos.
- Dos cierres simultáneos de la misma ronda: uno la cierra y el otro recibe 409 "La ronda ya está cerrada"; la fecha
  de cierre es la del primero.
- Un voto que llega mientras se cierra la ronda espera al cierre y recibe "La ronda ya está cerrada", sin cambiar el
  voto anterior; un voto confirmado antes del cierre entra en la revelación.
- Un integrante que votó y después fue quitado del proyecto figura en la ronda cerrada con su valor y ese valor cuenta
  para el rango (clarificación del 28/10); un integrante quitado que no votó no figura.
- Una ronda cerrada sin votos (solo posible con datos cargados fuera de la aplicación, como en pruebas de HU-05a)
  muestra a todos como "No votó", sin rango ni alerta, y la pantalla no falla.
- Con todas las rondas abiertas sin cerrar, la sección muestra "Aún no hay rondas cerradas".
- Los valores 1 y 89 (niveles 1 y 10) divergen; 1 y 3 (niveles 1 y 3) no divergen; 55 y 89 (niveles 9 y 10) no
  divergen.
- Recargar la página después de cerrar no reenvía el cierre ni repite el mensaje (patrón de HU-20b).
- Los títulos de las historias y los nombres de los integrantes se muestran como texto (escapados), también en el
  mensaje de cierre.
- El valor de un voto de una ronda abierta no aparece en ninguna parte de la respuesta, aunque en la misma pantalla
  haya rondas cerradas con sus valores revelados.
- La historia de una ronda cerrada no puede abrir otra ronda (HU-05a, "La historia ya tuvo una ronda de votación")
  hasta HU-18.
- Cerrar una ronda no modifica la historia (título, estado ni Story Points).

## Requisitos funcionales

- **FR-001**: Cada ronda abierta de la pantalla MUST ofrecer un formulario "Cerrar ronda" que envía el identificador
  de la ronda a `POST /planning-poker/cerrar`, dentro de la sección de HU-05a (proyecto activo e integrante
  identificado).
- **FR-002**: Cerrar una ronda abierta del proyecto activo con al menos un voto MUST cambiar su estado a "Cerrada" y
  registrar la fecha de cierre en una sola operación, responder 303 a `/planning-poker` y mostrar una sola vez "Ronda
  cerrada para la historia «título»".
- **FR-003**: Si la ronda está ausente, no es un identificador válido, no existe o es de otro proyecto, cerrar MUST
  responder 404 con "La ronda no existe" y MUST NOT modificar rondas ni votos. Esta comprobación MUST hacerse antes
  que las de FR-004 y FR-005.
- **FR-004**: Si la ronda ya está cerrada, cerrar MUST responder 409 con "La ronda ya está cerrada" y MUST NOT
  modificar la ronda (ni su fecha de cierre) ni sus votos.
- **FR-005**: Si la ronda abierta no tiene votos, cerrar MUST responder 409 con "La ronda no tiene votos" y la ronda
  MUST seguir abierta.
- **FR-006**: La regla de divergencia MUST ser una función pura del dominio que recibe los valores emitidos y
  devuelve el menor, el mayor y si hay divergencia según R1 y R2, para que HU-18 la reutilice.
- **FR-007**: La pantalla MUST mostrar la sección "Rondas cerradas" con las rondas cerradas del proyecto activo, de
  la más recientemente cerrada a la más antigua; sin rondas cerradas, MUST mostrar "Aún no hay rondas cerradas".
- **FR-008**: Cada ronda cerrada MUST mostrar el título de la historia, el número de ronda y la lista de R7: cada
  integrante con su valor o "No votó".
- **FR-009**: Con divergencia, cada ronda cerrada MUST mostrar la alerta "Divergencia: los votos van de X a Y. Debatan
  antes de acordar la estimación" con el formato de alerta de HU-20b (etiqueta de texto además del color); sin
  divergencia, MUST mostrar "Sin divergencia: los votos van de X a Y" y MUST NOT mostrar la alerta.
- **FR-010**: Los valores revelados, el rango y la alerta MUST ser los mismos para todos los integrantes del
  proyecto, sin importar quién cerró la ronda.
- **FR-011**: Los valores de los votos MUST leerse solo de rondas cerradas; la respuesta MUST NOT contener el valor
  del voto de otro integrante en una ronda abierta (FR-013 de HU-05a sigue vigente).
- **FR-012**: Un voto en una ronda cerrada MUST seguir respondiendo 409 con "La ronda ya está cerrada" sin cambiar el
  voto existente; un voto concurrente con el cierre MUST esperar y recibir ese mismo rechazo si el cierre se confirma
  primero.
- **FR-013**: Si falla el guardado del cierre, el sistema MUST responder 500 con "La operación no pudo completarse",
  MUST NOT mostrar un mensaje de éxito, la ronda MUST seguir abierta y sus valores MUST seguir ocultos.
- **FR-014**: Si falla la lectura de la pantalla, el sistema MUST mostrar la página de error de HU-20a y MUST NOT
  mostrar una pantalla incompleta.
- **FR-015**: Sin integrante identificado o sin proyecto activo MUST aplicarse la redirección de HU-20b, con regreso a
  Planning Poker, y MUST NOT cerrarse ninguna ronda.
- **FR-016**: El proyecto y el integrante MUST salir del contexto del servidor; todas las lecturas y escrituras MUST
  filtrarse por el proyecto activo.
- **FR-017**: Los títulos de historias y nombres de integrantes MUST mostrarse escapados como texto.
- **FR-018**: HU-05b MUST NOT agregar migraciones, guardar la divergencia, modificar la historia ni cambiar el
  comportamiento de abrir y votar de HU-05a.
- **FR-019**: Los textos nuevos MUST definirse junto a los de HU-05a; las respuestas de error MUST NOT incluir texto
  de consultas ni detalles internos.

## Entidades clave

- **Ronda de votación** (HU-05a): historia, número, estado ("Abierta" o "Cerrada"), quién la abrió, fecha de apertura
  y fecha de cierre. HU-05b la pasa a "Cerrada" y completa la fecha de cierre.
- **Voto** (HU-05a): valor de la serie de un integrante en una ronda. HU-05b lo revela cuando la ronda se cierra.
- **Rango de la ronda** (nuevo, no se guarda): menor valor, mayor valor y si hay divergencia, calculado con R1 y R2.
- **Integrante del proyecto** (HU-01, HU-20b): quien cierra y quien figura en la lista revelada.

## Criterios de éxito

- **SC-001**: En el 100% de las pruebas, el rango y la divergencia de cada ronda coinciden con las cuentas a mano de
  "Datos de ejemplo", incluido el límite exacto de dos niveles.
- **SC-002**: El 100% de los cierres válidos deja la ronda "Cerrada" con fecha de cierre, y ningún cierre rechazado
  (sin votos, ya cerrada, inexistente, de otro proyecto, sin integrante o con falla) modifica rondas ni votos.
- **SC-003**: Todos los integrantes ven, para la misma ronda cerrada, exactamente los mismos valores, rango y alerta.
- **SC-004**: En el 100% de las pantallas de prueba con rondas abiertas y cerradas a la vez, el valor de un voto ajeno
  de una ronda abierta no aparece en ninguna parte de la respuesta.
- **SC-005**: Un voto concurrente con el cierre nunca queda registrado en la ronda después de cerrarla.
- **SC-006**: El Scrum Master cierra una ronda y ve el resultado con una sola acción ("Cerrar ronda"), y la pantalla
  responde en menos de 500 ms con la base local, 20 rondas abiertas, 50 rondas cerradas y 10 integrantes.
- **SC-007**: Cada bloque tiene pruebas que fallaron antes del código (evidencia RED) y pasan después (evidencia
  GREEN), y la cobertura de `src/poker` queda en `cobertura.txt` con al menos 80%.

## Supuestos y decisiones

- No hay roles: cierra cualquier integrante identificado del proyecto (decisión del equipo; HU-28 lo restringirá).
- La identificación de HU-20b valida en cada solicitud que el integrante pertenezca al proyecto activo; HU-05b no
  repite esa comprobación.
- El cierre no guarda quién cerró la ronda: no hay columna para eso y la historia no agrega migraciones.
- El texto "Debatan antes de acordar la estimación" y el resto de los textos son los del Issue; se definen en el
  archivo único de mensajes.
- Las rondas cerradas de una historia siguen impidiendo abrir otra ronda (regla de HU-05a) hasta que HU-18 agregue las
  rondas siguientes.
- Los niveles salen de la serie que ya define HU-05a en el dominio de Planning Poker, sin duplicarla.
- El Issue no define el "otro proyecto" del escenario 10: las pruebas usan un proyecto «Beta» con su integrante
  Dani, una ronda abierta con un voto y una ronda cerrada sin votos. No cambia ningún dato ni cuenta de «Alfa».
- La sección "Rondas cerradas" va al final de la pantalla, después de las rondas abiertas.

## Trazabilidad

| Escenario | Requisitos | Éxito | Prueba de handler / BDD | Prueba de dominio / integración |
|---|---|---|---|---|
| 1 Revelación de votos | FR-001, FR-002, FR-007, FR-008, FR-016 | SC-002 | `TestHU05b_BotonCerrarRonda`, `TestHU05b_CerrarRonda`, BDD `TestBDD_HU05b_RevelacionDeVotos` | `TestService_CloseRound_OK`, `TestPostgresCloseRound_OK`, `TestPostgresPokerCatalog_RondasCerradas`, `TestPostgresPokerCatalog_VotosRevelados` |
| 2 Alerta de divergencia | FR-006, FR-009 | SC-001 | `TestHU05b_AlertaDeDivergencia`, BDD `TestBDD_HU05b_AlertaDeDivergencia` | `TestSpread` |
| 3 Sin divergencia en el límite | FR-006, FR-009 | SC-001 | `TestHU05b_SinDivergenciaEnElLimite`, BDD `TestBDD_HU05b_SinDivergenciaEnElLimite` | `TestSpread` |
| 4 Divergencia de tres niveles | FR-006, FR-009 | SC-001 | `TestHU05b_AlertaDeDivergencia`, BDD `TestBDD_HU05b_DivergenciaDeTresNiveles` | `TestSpread` |
| 5 Cerrar sin que todos voten | FR-002, FR-008, FR-009 | SC-001, SC-002 | `TestHU05b_CerrarSinTodosLosVotos`, `TestHU05b_SinDivergenciaEnElLimite`, BDD `TestBDD_HU05b_CerrarSinQueTodosVoten` | `TestSpread`, `TestPostgresCloseRound_SinTodosLosVotos`, `TestPostgresPokerCatalog_VotosRevelados` |
| 6 Ronda sin votos | FR-005 | SC-002 | `TestHU05b_RondaSinVotos`, BDD `TestBDD_HU05b_RondaSinVotos` | `TestCheckClose`, `TestSpread_Errores`, `TestPostgresCloseRound_SinVotos` |
| 7 Los votos se ven para todos | FR-010, FR-011 | SC-003, SC-004 | `TestHU05b_VotosParaTodos`, `TestHU05b_VotoAbiertoNuncaSeRevela`, BDD `TestBDD_HU05b_VotosParaTodos` | `TestPostgresPokerCatalog_VotosRevelados`, `TestPostgresPokerCatalog_SoloRondasCerradas` |
| 8 Votar en una ronda cerrada | FR-012 | SC-005 | `TestHU05b_VotarEnRondaCerrada`, BDD `TestBDD_HU05b_VotarEnRondaCerrada` | `TestPostgresCloseRound_VotoEsperaAlCierre` |
| 9 Cerrar dos veces | FR-004 | SC-002 | `TestHU05b_CerrarDosVeces`, BDD `TestBDD_HU05b_CerrarDosVeces` | `TestCheckClose`, `TestPostgresCloseRound_YaCerrada`, `TestPostgresCloseRound_Concurrente` |
| 10 Ronda inexistente o de otro proyecto | FR-003, FR-016 | SC-002 | `TestHU05b_RondaInexistente`, BDD `TestBDD_HU05b_RondaInexistente` | `TestService_CloseRound_Validaciones`, `TestPostgresCloseRound_Inexistente`, `TestPostgresCloseRound_OtroProyecto` |
| 11 Sin integrante identificado | FR-015 | SC-002 | `TestHU05b_SinIdentificarme`, BDD `TestBDD_HU05b_SinIdentificarme` | `TestService_CloseRound_Validaciones` |
| 12 Error al cerrar | FR-013, FR-019 | SC-002 | `TestHU05b_ErrorAlCerrar` | `TestPostgresCloseRound_FalloNoDejaCambios` |
| (Caso límite) Voto confirmado antes del cierre | FR-012 | SC-005 | — | `TestPostgresCloseRound_EsperaAlVotoEnCurso` |
| (Caso límite) Integrante quitado | FR-008 | SC-003 | — | `TestPostgresPokerCatalog_IntegranteQuitado` |
| (Caso límite) Sin rondas cerradas / cerrada sin votos | FR-007, FR-014 | — | `TestHU05b_SinRondasCerradas`, `TestHU05b_RondaCerradaSinVotos` | — |
| (HU-20a) Error al leer | FR-014 | — | `TestHU05a_ErrorAlLeer` (sigue en verde) | — |
| (HU-20b) Texto escapado | FR-017 | — | `TestHU05b_TextosEscapados` | — |
| (HU-20b) Recarga sin reenviar | FR-002 | SC-002 | `TestHU05b_RecargaSinReenviar` | — |
| Rendimiento de la pantalla | — | SC-006 | `TestHU05b_Rendimiento` | — |
| Regla reutilizable, sin migración ni cambios en HU-05a | FR-006, FR-018 | SC-007 | Pruebas de HU-05a en cada GREEN | `TestSpread_NoModificaLosValores`, `TestPostgresCloseRound_OK` (historia sin cambios), revisión en T026 |

### Ubicación y resultado de las pruebas (T029, 29/10/2026)

Todos los nombres de la tabla existen tal cual en el código. Resultado: todas en verde en
`evidencia-pruebas.txt` (`go test -count=1 -p 1 -v ./...`); la falla previa de cada bloque está en
`evidencia-red.txt` (secciones Divergencia, Cierre y Pantalla); cobertura de `src/poker` 89,3% en `cobertura.txt`.

| Archivo | Pruebas |
|---|---|
| `src/poker/spread_test.go` | `TestSpread`, `TestSpread_Errores`, `TestSpread_NoModificaLosValores` |
| `src/poker/round_test.go` | `TestCheckClose` |
| `src/poker/service_test.go` | `TestService_CloseRound_OK`, `TestService_CloseRound_Validaciones` |
| `src/poker/close_integration_test.go` | `TestPostgresCloseRound_OK`, `_SinTodosLosVotos`, `_SinVotos`, `_YaCerrada`, `_Inexistente`, `_OtroProyecto`, `_VotoEsperaAlCierre`, `_Concurrente`, `_EsperaAlVotoEnCurso`, `_FalloNoDejaCambios` |
| `src/web/postgres_poker_reveal_integration_test.go` | `TestPostgresPokerCatalog_RondasCerradas`, `_VotosRevelados`, `_IntegranteQuitado`, `_SoloRondasCerradas` |
| `src/web/poker_reveal_view_test.go` | `TestHU05b_AlertaDeDivergencia`, `_SinDivergenciaEnElLimite`, `_VotosParaTodos`, `_VotoAbiertoNuncaSeRevela`, `_SinRondasCerradas`, `_RondaCerradaSinVotos`, `_TextosEscapados`, `_Rendimiento` |
| `src/web/poker_close_handlers_test.go` | `TestHU05b_BotonCerrarRonda`, `_CerrarRonda`, `_CerrarSinTodosLosVotos`, `_RondaSinVotos`, `_VotarEnRondaCerrada`, `_CerrarDosVeces`, `_RondaInexistente`, `_SinIdentificarme`, `_ErrorAlCerrar`, `_RecargaSinReenviar` |
| `test/web_poker_reveal_bdd_test.go` | `TestBDD_HU05b_RevelacionDeVotos`, `_AlertaDeDivergencia`, `_SinDivergenciaEnElLimite`, `_DivergenciaDeTresNiveles`, `_CerrarSinQueTodosVoten`, `_RondaSinVotos`, `_VotosParaTodos`, `_VotarEnRondaCerrada`, `_CerrarDosVeces`, `_RondaInexistente`, `_SinIdentificarme` |
| `src/web/poker_view_test.go` (HU-05a, sin cambios) | `TestHU05a_ErrorAlLeer` |

Verificación manual (T028, 29/10/2026): los 12 escenarios con la aplicación en ejecución y los datos de «Alfa», OK.
El escenario 12 (falla al guardar) no se puede provocar a mano y queda cubierto por `TestHU05b_ErrorAlCerrar` y
`TestPostgresCloseRound_FalloNoDejaCambios`.
