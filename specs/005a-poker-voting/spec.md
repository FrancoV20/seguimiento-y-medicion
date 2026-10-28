# Especificación: HU-05a – Emisión de voto oculto en Planning Poker

**Rama**: `HU-05a`
**Creada**: 2026-10-27
**Estado**: Borrador
**Constitución**: v1.1.0
**Depende de**: HU-01, HU-02, HU-20a, HU-20b
**Habilita**: HU-05b (revelar votos), HU-18 (nueva ronda y estimación acordada)
**Prioridad**: Media-Alta (Should Have) · **Valor**: 85 · **Estimación**: 5 SP

**Entrada**: "HU-05a – Emisión de voto oculto en Planning Poker. Como Product Builder, quiero emitir mi
estimación de Story Points de forma oculta para una historia en discusión, para que el equipo pueda votar sin
influenciarse entre sí."

## Historia de usuario

Como Product Builder, quiero emitir mi estimación de Story Points de forma oculta para una historia en
discusión, para que el equipo pueda votar sin influenciarse entre sí.

> **Alcance**: HU-05a agrega la sección "Planning Poker": abrir la primera ronda de votación de una historia,
> votar con un valor de la serie de Fibonacci y cambiar el voto mientras la ronda está abierta. La pantalla
> muestra quién votó y el voto propio, nunca los valores de los demás. Revelar los votos y cerrar la ronda son de
> HU-05b; las rondas siguientes y la estimación acordada, de HU-18.
>
> **Integrante**: el voto y la apertura quedan a nombre del integrante identificado en HU-20b; el formulario no
> tiene campo de integrante. No hay roles: cualquier integrante identificado del proyecto abre rondas y vota.

## Clarifications

### Session 2026-10-27

- Q: ¿Para qué historias del proyecto se puede abrir una ronda (además de que nunca hayan tenido rondas)? → A:
  Para cualquier historia del proyecto activo, sin importar su estado ni si ya tiene Story Points.
- Q: ¿Qué historias lista el selector del formulario "Abrir ronda"? → A: Todas las historias del proyecto activo,
  incluidas las que ya tienen o tuvieron una ronda; el servidor rechaza esas con los escenarios 9 y 10.
- Q: En cada ronda abierta, ¿cómo se muestra quién votó? → A: Se listan todos los integrantes del proyecto con
  "Votó" o "Sin votar"; el valor aparece solo en la fila del integrante identificado.

## Sección SDD

### Intención

Permitir que cada integrante del proyecto emita una estimación de Story Points sin ver las de los demás, para
evitar el anclaje. HU-05a deja guardados los votos de la ronda abierta, que HU-05b revela y HU-18 usa para la
estimación acordada.

### Alcance

- Sección "Planning Poker" (`/planning-poker`) que exige proyecto activo e integrante identificado.
- Abrir la ronda 1 de una historia del proyecto que nunca tuvo rondas.
- Votar en una ronda abierta con un valor de la serie 1, 2, 3, 5, 8, 13, 21, 34, 55, 89; cambiar el voto
  reemplaza el anterior.
- Pantalla con las rondas abiertas del proyecto: historia, número de ronda, quién votó, el voto propio y el
  formulario de voto.
- Rechazos: valor vacío o fuera de la serie, ronda cerrada, ronda o historia inexistente, historia con ronda
  abierta, historia que ya tuvo una ronda, falla al guardar.

### Fuera de alcance

- Revelar votos y cerrar rondas (HU-05b).
- Detectar divergencia entre votos (HU-05b / HU-18).
- Abrir una nueva ronda para la misma historia (HU-18).
- Registrar la estimación acordada en la historia (HU-18): HU-05a no modifica los Story Points de la historia.
- Roles (Scrum Master, Product Owner) y permisos distintos por rol (limitación conocida, `docs/hu-20b-deuda-tecnica.md`).
- Historial de votos o rondas cerradas en la pantalla.

### Entradas

| Formulario | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `POST /planning-poker/abrir` | `historia` | identificador numérico | sí | Historia del proyecto activo |
| `POST /planning-poker/votar` | `ronda` | identificador numérico | sí | Ronda del proyecto activo |
| `POST /planning-poker/votar` | `valor` | número de la serie | sí | 1, 2, 3, 5, 8, 13, 21, 34, 55 u 89 |

El proyecto y el integrante no son campos: salen del contexto del servidor (HU-20b). Un campo de integrante o
de proyecto enviado en el formulario se ignora.

### Salidas esperadas

- Apertura válida: redirección 303 a `/planning-poker` con "Ronda abierta para la historia «título»" una sola vez.
- Primer voto: 303 a `/planning-poker` con "Voto registrado". Cambio de voto: 303 con "Voto actualizado".
- Valor vacío o fuera de la serie: 422 con la pantalla y el mensaje; nada cambia.
- Ronda cerrada, historia con ronda abierta o historia que ya tuvo una ronda: 409 con la pantalla y el mensaje.
- Ronda o historia inexistente: 404 con la pantalla y el mensaje.
- Falla al guardar: 500 con la pantalla y "La operación no pudo completarse".
- Falla al leer la pantalla: página de error de HU-20a.
- Sin integrante identificado o sin proyecto activo: redirección 303 de HU-20b.

### Reglas de negocio

- Serie válida: 1, 2, 3, 5, 8, 13, 21, 34, 55, 89 (la misma de los Story Points de HU-02).
- Una ronda está "Abierta" o "Cerrada". HU-05a solo crea rondas "Abierta" y siempre con número 1.
- Solo se abre una ronda para una historia que nunca tuvo rondas.
- Hay como máximo una ronda abierta por historia; varias historias pueden tener su ronda abierta a la vez.
- Un integrante tiene como máximo un voto por ronda; votar de nuevo reemplaza el valor anterior.
- Solo se vota en rondas abiertas.
- Mientras la ronda está abierta, el valor de un voto solo lo ve quien lo emitió.
- Se puede abrir una ronda para cualquier historia del proyecto activo, sin importar su estado ("Pendiente",
  "En Sprint", "En progreso", "Hecho") ni si ya tiene Story Points.
- Cada ronda abierta muestra a todos los integrantes del proyecto con "Votó" o "Sin votar".

### Restricciones

- Una migración nueva (tablas de rondas y votos); sin dependencias nuevas.
- No se editan archivos de otras historias salvo `src/cmd/server/main.go` para registrar la sección.
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript (HU-20a); la protección contra envíos de otro origen es el middleware
  global de HU-20b (los formularios no llevan token).
- Los mensajes se definen en un único archivo.

### Condiciones de error

| Condición | Mensaje | Respuesta |
|---|---|---|
| Voto sin valor | "Seleccioná un valor de la serie" | 422 |
| Valor fuera de la serie o no numérico | "El valor no pertenece a la serie de Fibonacci" | 422 |
| Votar en una ronda cerrada | "La ronda ya está cerrada" | 409 |
| Ronda inexistente, de otro proyecto o identificador inválido | "La ronda no existe" | 404 |
| Historia inexistente, de otro proyecto o identificador inválido | "La historia no existe" | 404 |
| Historia con una ronda abierta | "Ya hay una ronda abierta para esta historia" | 409 |
| Historia que ya tuvo una ronda (cerrada) | "La historia ya tuvo una ronda de votación" | 409 |
| Falla al guardar (abrir o votar) | "La operación no pudo completarse" | 500 |
| Falla al leer la pantalla | Página de error de HU-20a | 500 o 503 |
| Sin integrante identificado / sin proyecto activo | Mensaje de HU-20b | 303 |

## Escenarios BDD

### Bloque A – Abrir ronda

**Escenario 1: Abrir una ronda de votación**
- **Dado** que estoy identificado como integrante del proyecto activo y la historia «Login» del proyecto no
  tiene rondas
- **Cuando** elijo «Login» y abro una ronda de votación
- **Entonces** veo "Ronda abierta para la historia «Login»" y la ronda 1 de «Login» aparece abierta y sin votos

**Escenario 9: Historia con una ronda abierta**
- **Dado** que la historia «Login» tiene una ronda abierta
- **Cuando** intento abrir otra ronda para «Login»
- **Entonces** veo "Ya hay una ronda abierta para esta historia" y «Login» sigue teniendo una sola ronda

**Escenario 10: Historia que ya tuvo una ronda**
- **Dado** que la historia «Login» tuvo una ronda que ya está cerrada y no tiene rondas abiertas
- **Cuando** intento abrir una ronda para «Login»
- **Entonces** veo "La historia ya tuvo una ronda de votación" y no se crea ninguna ronda

### Bloque B – Votar y cambiar voto

**Escenario 2: Voto oculto registrado**
- **Dado** que la historia «Login» tiene una ronda abierta en la que todavía no voté
- **Cuando** elijo el valor 5 y confirmo mi voto
- **Entonces** veo "Voto registrado" y mi fila en la ronda indica "Votó" y muestra mi voto 5

**Escenario 3: Modificación de estimación en votación activa**
- **Dado** que en la ronda abierta de «Login» ya voté 5
- **Cuando** elijo el valor 8 y confirmo mi voto
- **Entonces** veo "Voto actualizado", mi voto es 8 y la ronda tiene un solo voto mío

**Escenario 4: Los votos de los demás permanecen ocultos**
- **Dado** que en la ronda abierta de «Login» Ana votó 13 y yo voté 3
- **Cuando** abro la pantalla de Planning Poker
- **Entonces** la fila de Ana indica "Votó" sin mostrar su valor, y mi fila indica "Votó" y muestra mi voto 3

### Bloque C – Errores y estados

**Escenario 5: Valor fuera de la serie**
- **Dado** que la historia «Login» tiene una ronda abierta
- **Cuando** se envía un voto con el valor 4
- **Entonces** veo "El valor no pertenece a la serie de Fibonacci" y no se registra ningún voto

**Escenario 6: Voto sin valor**
- **Dado** que la historia «Login» tiene una ronda abierta
- **Cuando** confirmo mi voto sin elegir un valor
- **Entonces** veo "Seleccioná un valor de la serie" y no se registra ningún voto

**Escenario 7: Votar en una ronda cerrada**
- **Dado** que la ronda de «Login» está cerrada
- **Cuando** se envía un voto para esa ronda
- **Entonces** veo "La ronda ya está cerrada" y los votos de la ronda no cambian

**Escenario 8: Ronda o historia inexistente**

Se escribe como dos escenarios simples, uno por acción:

*8a – Ronda inexistente*
- **Dado** que la ronda indicada no existe en el proyecto activo
- **Cuando** se envía un voto para esa ronda
- **Entonces** veo "La ronda no existe" y no se registra ningún voto

*8b – Historia inexistente*
- **Dado** que la historia indicada no existe en el proyecto activo
- **Cuando** se intenta abrir una ronda para esa historia
- **Entonces** veo "La historia no existe" y no se crea ninguna ronda

**Escenario 11: Error al guardar**
- **Dado** que la historia «Login» tiene una ronda abierta y el guardado falla
- **Cuando** elijo el valor 5 y confirmo mi voto
- **Entonces** veo "La operación no pudo completarse", no veo un mensaje de éxito y no se registra ningún voto

**Escenario 12: Votar sin identificarme**
- **Dado** que elegí el proyecto activo pero no me identifiqué como integrante
- **Cuando** envío un voto
- **Entonces** se me pide identificarme, no se registra ningún voto y, al identificarme, vuelvo a Planning Poker

### Bloque D – Pantalla y estados vacíos

**Escenario 13: Pantalla sin historias ni rondas**
- **Dado** que el proyecto activo no tiene historias
- **Cuando** abro Planning Poker
- **Entonces** veo "Aún no hay historias" y "No hay rondas abiertas", y no se ofrece abrir una ronda

### Casos límite

- El integrante sale siempre de la identificación del servidor; un campo de integrante en el formulario se ignora.
- Un `historia` o `ronda` vacío, no numérico, negativo, con ceros a la izquierda o mayor que el máximo se trata
  como inexistente (404) sin consultar la base.
- Una historia o ronda de otro proyecto se trata como inexistente (404): no revela datos de otros proyectos.
- Un `valor` no numérico ("abc", "5.0", " 5") se trata como fuera de la serie (422).
- Un valor rechazado conserva el voto anterior del integrante, si lo tenía.
- Al votar, el valor se valida antes que la ronda: un voto sin valor sobre una ronda inexistente responde 422
  "Seleccioná un valor de la serie". En ningún caso se modifican datos.
- Dos aperturas simultáneas de la misma historia: una abre la ronda y la otra recibe 409 "Ya hay una ronda
  abierta para esta historia".
- Votos simultáneos de distintos integrantes en la misma ronda se registran todos.
- Si la ronda se cierra (HU-05b) mientras alguien envía un voto, el voto se registra antes del cierre o recibe
  409 "La ronda ya está cerrada"; nunca queda un voto en una ronda después de cerrarla.
- Los títulos de las historias y los nombres de los integrantes se muestran como texto (escapados).
- El valor de un voto ajeno no aparece en ninguna parte de la respuesta: ni texto, ni atributos, ni campos
  ocultos, ni el valor preseleccionado del formulario.
- Recargar la página después de votar no reenvía el voto ni repite el mensaje (patrón de HU-20b).
- Las rondas cerradas no se muestran en la pantalla de HU-05a.
- Un integrante agregado al proyecto mientras hay una ronda abierta aparece en ella como "Sin votar"; los votos
  de un integrante quitado del proyecto se conservan pero no se muestran (la lista sale de los integrantes
  actuales).
- Una historia en "Hecho" o que ya tiene Story Points se puede votar igual (decisión del equipo).
- La identificación pasa a otro integrante: la pantalla muestra el voto propio del nuevo integrante, no el del
  anterior.

## Requisitos funcionales

- **FR-001**: El sistema MUST ofrecer la sección "Planning Poker" en `/planning-poker`, que exige proyecto activo
  e integrante identificado.
- **FR-002**: La pantalla MUST mostrar el formulario para abrir una ronda con un selector que lista todas las
  historias del proyecto activo, cualquiera sea su estado o estimación, incluidas las que ya tienen o tuvieron una
  ronda (las rechazan FR-006 y FR-007).
- **FR-003**: La pantalla MUST mostrar cada ronda abierta del proyecto con el título de la historia, el número de
  ronda, la lista de todos los integrantes del proyecto con "Votó" o "Sin votar", el voto propio (si existe) en la
  fila del integrante identificado y un formulario de voto con los diez valores de la serie.
- **FR-004**: Abrir una ronda válida MUST crear la ronda número 1 de la historia en estado "Abierta", registrar
  al integrante identificado como quien la abrió, responder 303 a `/planning-poker` y mostrar una sola vez
  "Ronda abierta para la historia «título»".
- **FR-005**: Si la historia está ausente, no es un identificador válido, no existe o es de otro proyecto, abrir
  MUST responder 404 con "La historia no existe" y MUST NOT crear rondas.
- **FR-006**: Si la historia tiene una ronda abierta, abrir MUST responder 409 con "Ya hay una ronda abierta para
  esta historia" y MUST NOT crear rondas.
- **FR-007**: Si la historia tuvo una ronda (cerrada) y no tiene abiertas, abrir MUST responder 409 con "La
  historia ya tuvo una ronda de votación" y MUST NOT crear rondas.
- **FR-008**: Un voto con un valor de la serie en una ronda abierta MUST registrarse a nombre del integrante
  identificado; si es su primer voto en la ronda, MUST responder 303 y mostrar "Voto registrado".
- **FR-009**: Si el integrante ya votó en la ronda, el nuevo voto MUST reemplazar el valor anterior (un solo voto
  por integrante y ronda), responder 303 y mostrar "Voto actualizado".
- **FR-010**: Un voto sin valor MUST responder 422 con "Seleccioná un valor de la serie"; un valor que no es de
  la serie o no es numérico MUST responder 422 con "El valor no pertenece a la serie de Fibonacci". En ambos
  casos MUST NOT modificar votos y el voto anterior del integrante se conserva.
- **FR-011**: Un voto en una ronda cerrada MUST responder 409 con "La ronda ya está cerrada" y MUST NOT modificar
  votos.
- **FR-012**: Un voto con una ronda ausente, con identificador inválido, inexistente o de otro proyecto MUST
  responder 404 con "La ronda no existe" y MUST NOT modificar votos.
- **FR-013**: Mientras una ronda está abierta, la respuesta MUST NOT contener el valor del voto de otro integrante
  en ninguna forma (texto, atributos, campos ocultos o valor preseleccionado); el valor propio MUST mostrarse solo
  a quien lo emitió.
- **FR-014**: Si falla el guardado al abrir o al votar, el sistema MUST responder 500 con "La operación no pudo
  completarse", MUST NOT mostrar un mensaje de éxito y MUST NOT dejar cambios parciales.
- **FR-015**: Si falla la lectura de la pantalla, el sistema MUST mostrar la página de error de HU-20a y MUST NOT
  mostrar una pantalla incompleta.
- **FR-016**: Sin integrante identificado o sin proyecto activo MUST aplicarse la redirección de HU-20b, con
  regreso a Planning Poker, y MUST NOT registrarse votos ni rondas.
- **FR-017**: Si el proyecto no tiene historias, la pantalla MUST mostrar "Aún no hay historias" y MUST NOT
  mostrar el formulario para abrir una ronda; si no hay rondas abiertas, MUST mostrar "No hay rondas abiertas".
- **FR-018**: El proyecto y el integrante MUST salir del contexto del servidor; todas las lecturas y escrituras
  MUST filtrarse por el proyecto activo.
- **FR-019**: Abrir una ronda MUST bloquear la historia durante la operación, de modo que dos aperturas
  simultáneas dejen una sola ronda abierta; votar MUST tomar la ronda con un bloqueo compartido, de modo que el
  cierre de HU-05b se serialice con los votos.
- **FR-020**: Los títulos de historias y nombres de integrantes MUST mostrarse escapados como texto.
- **FR-021**: HU-05a MUST NOT cerrar rondas, revelar votos ni modificar los Story Points de la historia.
- **FR-022**: La serie de votación MUST ser exactamente la de Story Points válidos de HU-02.
- **FR-023**: Las respuestas de error MUST NOT incluir texto de consultas ni detalles internos.

## Entidades clave

- **Ronda de votación**: pertenece a una historia del proyecto; tiene número (1 en HU-05a), estado ("Abierta" o
  "Cerrada"), integrante que la abrió, fecha de apertura y fecha de cierre (solo si está cerrada).
- **Voto**: valor de la serie emitido por un integrante en una ronda; uno por integrante y ronda, con la fecha de
  su última modificación.
- **Historia de usuario** (HU-02): la que se estima; HU-05a no la modifica.
- **Integrante del proyecto** (HU-01, HU-20b): quien abre rondas y vota.

## Criterios de éxito

- **SC-001**: El 100% de las aperturas válidas deja exactamente una ronda abierta, número 1, para esa historia.
- **SC-002**: Si N integrantes votan una o más veces en una ronda, la ronda termina con exactamente N votos, cada
  uno con el último valor elegido por su integrante.
- **SC-003**: En el 100% de las pantallas de prueba con una ronda abierta, el valor de un voto ajeno no aparece en
  la fila de ese integrante ni en ninguna otra parte de la respuesta fuera del selector de la serie.
- **SC-004**: Ningún envío rechazado (valor vacío o fuera de la serie, ronda cerrada o inexistente, historia
  inexistente, con ronda abierta o que ya tuvo una) modifica rondas o votos.
- **SC-005**: Dos aperturas simultáneas de la misma historia terminan con una apertura exitosa y otra rechazada
  con 409, y una sola ronda abierta.
- **SC-006**: Ninguna falla de guardado muestra un mensaje de éxito ni deja cambios parciales.
- **SC-007**: Un integrante emite o cambia su voto con dos acciones (elegir el valor y confirmar), y la pantalla
  responde en menos de 500 ms con la base local, 50 historias, 20 rondas abiertas y 10 integrantes.
- **SC-008**: Cada escenario tiene una prueba que falló antes del código (evidencia RED) y pasa después
  (evidencia GREEN), y la cobertura de `src/poker` queda en `cobertura.txt` con al menos 80%.

## Supuestos y decisiones

- No hay roles: abre ronda y vota cualquier integrante identificado del proyecto (decisión del equipo).
- La identificación de HU-20b valida en cada solicitud que el integrante pertenezca al proyecto activo; HU-05a no
  repite esa comprobación.
- La serie se define en el paquete de Planning Poker y una prueba comprueba, con la validación de HU-02, que
  coincide con los Story Points válidos.
- HU-05a no crea rondas cerradas; las pruebas de los escenarios 7 y 10 las preparan directamente en la base.
- Las rondas cerradas no se muestran en esta pantalla; HU-05b decide cómo mostrarlas.
- Los textos de los mensajes son una propuesta del equipo y se definen en un único archivo.

## Trazabilidad

| Escenario | Requisitos | Éxito | Prueba de handler / BDD | Prueba de dominio / integración |
|---|---|---|---|---|
| 1 Abrir una ronda | FR-001, FR-002, FR-004, FR-018 | SC-001 | `TestHU05a_AbrirRonda`, `TestHU05a_SelectorTodasLasHistorias`, BDD `TestBDD_HU05a_AbrirRonda` | `TestCheckOpen_SinRondas`, `TestService_OpenRound_OK`, `TestPostgresOpenRound_OK`, `TestPostgresPokerCatalog_Historias`, `TestPostgresPokerCatalog_RondasAbiertas` |
| 2 Voto oculto registrado | FR-003, FR-008, FR-018 | SC-002 | `TestHU05a_VotoRegistrado`, BDD `TestBDD_HU05a_VotoOculto` | `TestParseValue_Serie`, `TestCheckVote_Abierta`, `TestPostgresCastVote_Nuevo`, `TestPostgresCastVote_DosIntegrantes`, `TestPostgresPokerCatalog_Participantes` |
| 3 Cambio de voto | FR-009 | SC-002 | `TestHU05a_VotoActualizado`, BDD `TestBDD_HU05a_CambioDeVoto` | `TestService_Vote_DevuelveUpdatedYPropagaErrores`, `TestPostgresCastVote_Reemplaza` |
| 4 Votos ajenos ocultos | FR-003, FR-013 | SC-003 | `TestHU05a_VotosAjenosOcultos`, `TestHU05a_CambioDeIdentificacion`, BDD `TestBDD_HU05a_VotosOcultos` | `TestPostgresPokerCatalog_SinValoresAjenos`, `TestPostgresPokerCatalog_VotoPropio`, `TestPokerViewModel_SinCampoDeValorAjeno` |
| 5 Valor fuera de la serie | FR-010, FR-022 | SC-004 | `TestHU05a_ValorFueraDeSerie` | `TestParseValue_FueraDeSerie`, `TestSeries_CoincideConStoryPoints`, `TestMigracionPoker_Restricciones` |
| 6 Voto sin valor | FR-010 | SC-004 | `TestHU05a_VotoSinValor` | `TestParseValue_Vacio`, `TestService_Vote_ValorVacioAntesDelAlmacen` |
| 7 Ronda cerrada | FR-011, FR-019 | SC-004 | `TestHU05a_RondaCerrada`, BDD `TestBDD_HU05a_RondaCerrada` | `TestCheckVote_Cerrada`, `TestPostgresCastVote_RondaCerrada`, `TestPostgresCastVote_EsperaAlCierre` |
| 8a Ronda inexistente | FR-012, FR-018 | SC-004 | `TestHU05a_RondaInexistente` | `TestService_Vote_RondaYIntegrante`, `TestPostgresCastVote_Inexistente`, `TestPostgresCastVote_OtroProyecto` |
| 8b Historia inexistente | FR-005, FR-018 | SC-004 | `TestHU05a_HistoriaInexistente` | `TestService_OpenRound_Validaciones`, `TestPostgresOpenRound_Inexistente`, `TestPostgresOpenRound_OtroProyecto` |
| 9 Historia con ronda abierta | FR-006, FR-019 | SC-004, SC-005 | `TestHU05a_HistoriaConRondaAbierta`, BDD `TestBDD_HU05a_HistoriaConRondaAbierta` | `TestCheckOpen_Abierta`, `TestPostgresOpenRound_ConRondaAbierta`, `TestPostgresOpenRound_Concurrente` |
| 10 Historia que ya tuvo ronda | FR-007 | SC-004 | `TestHU05a_HistoriaYaTuvoRonda` | `TestCheckOpen_Cerrada`, `TestPostgresOpenRound_YaTuvoRonda` |
| 11 Error al guardar | FR-014, FR-023 | SC-006 | `TestHU05a_ErrorAlGuardar` | `TestPostgresCastVote_FalloNoDejaCambios` |
| 12 Sin identificarme | FR-016 | SC-004 | `TestHU05a_SinIdentificarme`, BDD `TestBDD_HU05a_SinIdentificarme` | — |
| 13 Sin historias ni rondas | FR-017 | — | `TestHU05a_PantallaVacia`, BDD `TestBDD_HU05a_PantallaVacia` | `TestPostgresPokerCatalog_Vacio` |
| (HU-20a) Error al leer | FR-015 | — | `TestHU05a_ErrorAlLeer` | — |
| (HU-20b) Texto escapado | FR-020 | — | `TestHU05a_TituloEscapado` | — |
| (HU-20b) Recarga sin reenviar | FR-008 | SC-002 | `TestHU05a_RecargaSinReenviar` | — |
| Rendimiento de la pantalla | — | SC-007 | `TestHU05a_Rendimiento` | — |
| (HU-05b/HU-18) Sin efectos fuera de alcance | FR-021 | — | — | `TestPostgresOpenRound_NoModificaHistoria` |
