# Especificación: HU-16 – Consulta de Sprints Anteriores

**Rama**: `HU-16`
**Creada**: 2026-10-28
**Estado**: Implementada (pendiente de revisión en el Pull Request)
**Constitución**: v1.1.0
**Depende de**: HU-07, HU-15, HU-20a, HU-20b (verificación manual: HU-23 y HU-14 para armar y completar Sprints)
**Prioridad**: Media (Should Have) · **Valor**: 65 · **Estimación**: 3 SP (sugerida)

**Entrada**: "HU-16 / Consulta de Sprints Anteriores. Como Scrum Master, quiero consultar los Sprints finalizados y
el detalle de cada uno, para analizar el desempeño pasado del equipo."

## Historia de usuario

Como Scrum Master, quiero consultar los Sprints finalizados y el detalle de cada uno, para analizar el desempeño
pasado del equipo.

> **Alcance**: HU-16 agrega la sección "Sprints anteriores", de solo lectura, con dos pantallas: la lista de los
> Sprints finalizados del proyecto activo, del más reciente al más antiguo, y el detalle de cada uno (objetivo,
> fechas, velocidad, historias completadas y no completadas).
>
> **No recalcula nada**: las historias y sus Story Points al cerrar salen del registro de cierre que guarda HU-15; la
> velocidad, de HU-07. Lo nuevo es una lectura de solo lectura y la pantalla.
>
> **Por qué una sección nueva**: "Sprints" (HU-23) muestra el Sprint en curso y "Métricas" (HU-25), la velocidad y la
> desviación; ninguna lista las historias de un Sprint ya cerrado.

## Clarifications

### Session 2026-10-28

Decisiones del equipo registradas antes de `/speckit-clarify` (no se vuelven a preguntar):

- Q: ¿Cómo se ofrece la pantalla? → A: Sección "Sprints anteriores" en `/sprints-anteriores`, que exige proyecto
  activo, con dos rutas en el mismo registro: `GET /sprints-anteriores` (lista) y `GET /sprints-anteriores/{id}`
  (detalle). Cualquier otro método → 405 de HU-20a. Sin formularios ni JavaScript.
- Q: ¿Qué muestra la lista? → A: Los Sprints "Finalizado" del proyecto activo, por número descendente, cada uno con
  "Sprint N", su objetivo, sus fechas (dd/mm/aaaa) y un enlace al detalle. El Sprint activo no figura. Sin
  finalizados: "Aún no hay Sprints anteriores".
- Q: ¿Cuándo el detalle responde 404? → A: Un `{id}` inválido, inexistente, de otro proyecto o de un Sprint que no
  está "Finalizado" → 404 de HU-20a, decidido antes de calcular nada.
- Q: ¿Qué muestra el detalle? → A: "Sprint N", objetivo, fechas, "Velocidad: N Story Points" y dos tablas,
  Completadas y No completadas, con título, Story Points al cerrar ("Sin estimar" si faltan) y estado al cerrar; las
  no completadas llevan "volvió al Backlog". Cada tabla se ordena por historia (orden de creación). Sin no
  completadas: "Todas las historias se completaron"; sin completadas: "Ninguna historia se completó". Sin filas en el
  registro de cierre: "No hay registro de las historias de este Sprint", y la velocidad se muestra igual.
- Q: ¿De dónde salen los datos? → A: Las historias, del registro de cierre de HU-15 unido a las historias (para el
  título), leído junto con el Sprint en una lectura consistente. La velocidad, del cálculo de métricas de HU-07, como
  en HU-25; ese cálculo vuelve a guardar los mismos valores en la tabla de métricas del Sprint.
- Q: ¿Qué pasa si falla una lectura? → A: Cualquier falla de la lectura del Sprint o del cálculo de HU-07 → página de
  error de HU-20a (500, o 503 si la base no responde), sin datos parciales. Sin proyecto activo, la redirección de
  HU-20b con regreso a la dirección pedida.
- Q: ¿Qué archivos se tocan? → A: No se editan archivos de HU-07, HU-15, HU-25 ni de otras historias. El único archivo
  compartido es el arranque del servidor, al que se suma un bloque de registro al final.
- Q: ¿Qué queda afuera? → A: Desviación y calidad (HU-25 y HU-10), exportar (HU-09) y modificar Sprints o historias.

## Sección SDD

### Intención

Que el Scrum Master revise cómo le fue al equipo en cada Sprint ya cerrado: qué se propuso, cuánto completó
(velocidad) y qué historias quedaron sin terminar y volvieron al Backlog.

### Alcance

- Sección "Sprints anteriores" del proyecto activo, de solo lectura.
- Lista de Sprints finalizados, del más reciente al más antiguo, con número, objetivo, fechas y enlace al detalle.
- Detalle de un Sprint finalizado: objetivo, fechas, velocidad, historias completadas y no completadas, con sus
  Story Points y su estado al cerrar.
- Mensajes para la lista vacía, el Sprint sin historias completadas, el Sprint con todas completadas y el Sprint sin
  registro de cierre.

### Fuera de alcance

- Desviación de esfuerzo y métricas de calidad (HU-25, HU-10); gráficos (HU-08).
- Exportar el Sprint a PDF (HU-09).
- Modificar Sprints, historias, Story Points o el registro de cierre.
- Reabrir un Sprint o volver a cerrarlo (HU-15).

### Entradas

| Origen | Dato | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `GET /sprints-anteriores` | — | — | — | La lista no recibe parámetros |
| `GET /sprints-anteriores/{id}` | `id` | entero entre 1 y el máximo de la base | sí | Inválido → 404 sin consultar la base |
| Contexto (HU-20b) | proyecto activo | identificador | sí | Nunca sale de la dirección ni de un formulario |

### Salidas esperadas

- Lista: 200 con los Sprints finalizados (número, objetivo, fechas y enlace), o "Aún no hay Sprints anteriores".
- Detalle: 200 con "Sprint N", objetivo, fechas, "Velocidad: N Story Points" y las dos tablas o sus mensajes.
- `{id}` inválido, inexistente, de otro proyecto o de un Sprint en curso: la página 404 de HU-20a.
- Falla al leer o al calcular la velocidad: la página de error de HU-20a (500 o 503), sin datos parciales.
- Sin proyecto activo: redirección 303 de HU-20b a elegir proyecto, con regreso a la dirección pedida.
- Cualquier método distinto de `GET` en las dos direcciones: 405 de HU-20a.

### Reglas de negocio

- R1: Sprint anterior = Sprint del proyecto activo en estado "Finalizado". El Sprint activo no es anterior.
- R2: la lista se ordena por número de Sprint, de mayor a menor.
- R3: el detalle existe solo para un Sprint anterior del proyecto activo; cualquier otro `{id}` es 404, y el 404 se
  decide antes de calcular la velocidad.
- R4: las historias del detalle son las filas del registro de cierre del Sprint (HU-15), con el título actual de la
  historia, los Story Points y el estado que tenía al cerrar. Completada = estado al cerrar "Hecho"; no completada =
  cualquier otro ("En Sprint" o "En progreso"), que HU-15 devolvió al Backlog.
- R5: velocidad = la de HU-07 para ese Sprint. No se suma en la pantalla. Con un registro de cierre hecho por HU-15,
  coincide con la suma de los Story Points de las completadas.
- R6: cada tabla se ordena por historia, en el orden en que se crearon.
- R7: sin no completadas → "Todas las historias se completaron" en lugar de esa tabla; sin completadas → "Ninguna
  historia se completó" en lugar de esa tabla; sin filas en el registro → "No hay registro de las historias de este
  Sprint" en lugar de las dos tablas, con la velocidad igual.
- R8: Story Points faltantes → "Sin estimar".
- R9: el Sprint y sus historias se leen juntos, de modo que sean coherentes entre sí.
- R10: la consulta no modifica Sprints, historias ni el registro de cierre.

### Restricciones

- Sin migración ni tablas nuevas; sin dependencias nuevas; sin JavaScript.
- No se editan archivos de HU-07, HU-15, HU-25 ni de otras historias. El único archivo compartido que se edita es el
  arranque del servidor (se suma un bloque de registro).
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- Los textos de la pantalla se definen en un único lugar.

### Condiciones de error

| Condición | Resultado | Respuesta |
|---|---|---|
| Proyecto sin Sprints finalizados | "Aún no hay Sprints anteriores" | 200 |
| `{id}` vacío, no numérico, cero, negativo, con ceros a la izquierda o muy grande | Página 404 de HU-20a, sin consultar la base | 404 |
| `{id}` inexistente, de otro proyecto o de un Sprint en curso | Página 404 de HU-20a, sin calcular la velocidad | 404 |
| Sprint finalizado sin registro de cierre | "No hay registro de las historias de este Sprint" y la velocidad | 200 |
| Falla al leer la lista, el Sprint o sus historias, o al calcular la velocidad | Página de error de HU-20a, sin datos parciales | 500 o 503 |
| Sin proyecto activo | Mensaje de HU-20b | 303 |
| Método distinto de `GET` | Página de HU-20a | 405 |

## Escenarios BDD

Los datos de ejemplo son los del Issue (los mismos de HU-09); el escenario 5 usa un proyecto propio (ver "Datos de
ejemplo").

### Bloque A – Lista

**Escenario 1: Listado de Sprints finalizados**
- **Dado** el proyecto «Alfa», con los Sprints 1 y 2 finalizados
- **Cuando** abro "Sprints anteriores"
- **Entonces** veo el Sprint 2 y después el Sprint 1, cada uno con su número, su objetivo y sus fechas

**Escenario 2: El Sprint activo no aparece**
- **Dado** que el Sprint 3 de «Alfa» está en curso
- **Cuando** abro "Sprints anteriores"
- **Entonces** el Sprint 3 no figura en la lista

**Escenario 6: Sin Sprints finalizados**
- **Dado** el proyecto «Beta», sin Sprints finalizados
- **Cuando** abro "Sprints anteriores"
- **Entonces** veo "Aún no hay Sprints anteriores"

### Bloque B – Detalle

**Escenario 3: Detalle de un Sprint**
- **Dado** el Sprint 2 finalizado de «Alfa»
- **Cuando** lo selecciono
- **Entonces** veo su objetivo, sus fechas, la velocidad en 10 Story Points, las historias completadas («Login» 5,
  «Tablero» 3 y «Perfil» 2) y las no completadas («Reportes» 8, "En progreso" al cerrar, volvió al Backlog)

**Escenario 4: Sprint con todas las historias completadas**
- **Dado** el Sprint 1 finalizado de «Alfa»
- **Cuando** lo selecciono
- **Entonces** veo «Alta» entre las completadas, la velocidad en 3 Story Points y "Todas las historias se
  completaron" en lugar de la lista de no completadas

**Escenario 5: Sprint sin historias completadas**
- **Dado** un Sprint finalizado en el que ninguna historia llegó a "Hecho"
- **Cuando** lo selecciono
- **Entonces** veo la velocidad en 0 Story Points, "Ninguna historia se completó" y la lista de las no completadas

### Bloque C – Solo lectura, direcciones y errores

**Escenario 7: Solo lectura**
- **Dado** el detalle del Sprint 2 finalizado
- **Cuando** lo veo o se envía una modificación a su dirección
- **Entonces** no hay ningún formulario ni acción para modificarlo y el envío se rechaza con la página 405 de la
  aplicación

**Escenario 8: Sprint inexistente, en curso o de otro proyecto**
- **Dado** un Sprint que no existe, que está en curso o que pertenece a otro proyecto
- **Cuando** abro su detalle
- **Entonces** veo la página 404 de la aplicación

**Escenario 9: Error al consultar**
- **Dado** que consulto Sprints anteriores
- **Cuando** ocurre un error al leer los datos
- **Entonces** veo la página de error de la aplicación, sin detalles internos y sin datos parciales

### Datos de ejemplo

Proyecto «Alfa», del 18/08/2026 al 30/11/2026 (como en HU-09).

| Sprint | Estado | Objetivo | Fechas |
|---|---|---|---|
| 1 | Finalizado | Dar de alta usuarios | 18/08/2026 al 31/08/2026 |
| 2 | Finalizado | Gestionar perfiles y reportes | 01/09/2026 al 14/09/2026 |
| 3 | Activo (en curso) | Exportar e informar | 15/09/2026 al 28/09/2026 |

Historias, en el orden en que se crearon:

| Historia | Sprint | Story Points | Estado al cerrar (registro de HU-15) | Estado actual | Asignación actual |
|---|---|---|---|---|---|
| «Alta» | 1 | 3 | Hecho | Hecho | Sprint 1 |
| «Login» | 2 | 5 | Hecho | Hecho | Sprint 2 |
| «Tablero» | 2 | 3 | Hecho | Hecho | Sprint 2 |
| «Perfil» | 2 | 2 | Hecho | Hecho | Sprint 2 |
| «Reportes» | 2 | 8 | En progreso | Pendiente (volvió al Backlog) | ninguna |
| «Exportar» | 3 | 5 | — (Sprint en curso) | En progreso | Sprint 3 |
| «Ayuda» | 3 | Sin estimar | — (Sprint en curso) | En Sprint | Sprint 3 |

Proyecto «Beta» (en las pruebas, del 01/10/2026 al 31/12/2026): sin Sprints ni historias.

Proyecto «Gamma» (propio de las pruebas, del 01/09/2026 al 31/12/2026; el Issue no da datos para el escenario 5):

| Sprint | Estado | Objetivo | Fechas | Registro de cierre | Asignaciones actuales |
|---|---|---|---|---|---|
| 1 | Finalizado | Migrar datos | 01/09/2026 al 14/09/2026 | «Migración» 5, En progreso; «Respaldo» Sin estimar, En Sprint | ninguna (las dos volvieron al Backlog) |
| 2 | Finalizado | Auditar accesos | 15/09/2026 al 28/09/2026 | sin filas (cargado antes de HU-15) | «Auditoría» 2, Hecho |

Cuentas:

- Lista de «Alfa»: finalizados 1 y 2; por número descendente → "Sprint 2" ("Gestionar perfiles y reportes", "Del
  01/09/2026 al 14/09/2026") y después "Sprint 1" ("Dar de alta usuarios", "Del 18/08/2026 al 31/08/2026"). El 3
  está "Activo" → no figura.
- Sprint 2: registro con 4 filas. Completadas («Hecho»): «Login» 5, «Tablero» 3 y «Perfil» 2 → 5 + 3 + 2 = 10. No
  completadas: «Reportes» 8, "En progreso", volvió al Backlog. HU-07 suma las "Hecho" que siguen asignadas al Sprint 2
  («Login», «Tablero» y «Perfil») → velocidad 10 = suma de las completadas del registro. Planificados: 10 + 8 = 18
  (no se muestra; HU-10).
- Sprint 1: registro con 1 fila, «Alta» 3, "Hecho". HU-07: «Alta» asignada y "Hecho" → velocidad 3 = 3. Sin no
  completadas → "Todas las historias se completaron".
- «Gamma», Sprint 1: registro con 2 filas, ninguna "Hecho" → "Ninguna historia se completó"; no completadas
  «Migración» 5 y «Respaldo» "Sin estimar", las dos con "volvió al Backlog". HU-15 borró sus asignaciones → HU-07 no
  ve historias → velocidad 0 = suma vacía.
- «Gamma», Sprint 2: sin filas en el registro → "No hay registro de las historias de este Sprint"; HU-07 ve «Auditoría»
  asignada y "Hecho" con 2 → "Velocidad: 2 Story Points".
- Escenario 8: el Sprint 3 de «Alfa» está "Activo" → 404; un Sprint de «Gamma» pedido con «Alfa» activo → 404; el
  identificador 2147483647 no existe → 404.
- «Beta»: 0 Sprints finalizados → "Aún no hay Sprints anteriores".

### Casos límite

- `{id}` vacío, "abc", "0", "-1", "01", "+2" o mayor que 2147483647 → 404 sin consultar la base (regla de HU-20b para
  identificadores).
- Un Sprint "Finalizado" sin filas en el registro de cierre (datos anteriores a HU-15 o cargados a mano) muestra "No
  hay registro de las historias de este Sprint" y la velocidad de HU-07.
- Una historia "Hecho" sin Story Points en un Sprint cargado a mano hace fallar el cálculo de HU-07: se muestra la
  página de error (500), no un detalle sin velocidad. HU-15 no deja cerrar un Sprint así.
- Una historia no completada que después entró a otro Sprint, o que se terminó en otro Sprint, sigue mostrándose en
  este con su estado al cerrar y "volvió al Backlog": el registro no cambia.
- El título que se muestra es el actual de la historia; si se editó después del cierre, se ve el nuevo.
- "Velocidad: 1 Story Points": la unidad no cambia con el número.
- Dos Sprints finalizados con las mismas fechas se ordenan igual por número.
- Los títulos de las historias y los objetivos se muestran escapados.
- La dirección del detalle de un Sprint de otro proyecto no revela que existe: da el mismo 404 que uno inexistente.
- Abrir el detalle vuelve a guardar los mismos valores de métricas del Sprint (efecto de HU-07); no cambia Sprints,
  historias ni el registro.
- Un envío de otro origen a esas direcciones lo rechaza antes la protección de HU-20b (403); el 405 se ve con envíos
  del mismo origen.

## Requisitos funcionales

- **FR-001**: El sistema MUST ofrecer la sección "Sprints anteriores" en `/sprints-anteriores`, que exige proyecto
  activo, con las direcciones `/sprints-anteriores` y `/sprints-anteriores/{id}`, y MUST responder solo a `GET` (y
  `HEAD`, que el enrutador de HU-20a atiende igual); cualquier otro método responde 405.
- **FR-002**: La lista MUST mostrar solo los Sprints "Finalizado" del proyecto activo, por número descendente, cada uno
  con "Sprint N", su objetivo, sus fechas en dd/mm/aaaa y un enlace a su detalle.
- **FR-003**: La lista MUST NOT mostrar el Sprint activo.
- **FR-004**: Sin Sprints finalizados, la lista MUST mostrar "Aún no hay Sprints anteriores".
- **FR-005**: El detalle MUST mostrar "Sprint N", el objetivo, las fechas en dd/mm/aaaa y "Velocidad: N Story Points".
- **FR-006**: El detalle MUST mostrar las historias del registro de cierre de HU-15 en dos tablas, Completadas y No
  completadas, con título, Story Points al cerrar ("Sin estimar" si faltan) y estado al cerrar; cada fila de No
  completadas MUST decir "volvió al Backlog". Cada tabla MUST ordenarse por historia, en orden de creación.
- **FR-007**: Sin no completadas, el detalle MUST mostrar "Todas las historias se completaron" en lugar de esa tabla;
  sin completadas, "Ninguna historia se completó" en lugar de esa tabla.
- **FR-008**: Sin filas en el registro de cierre, el detalle MUST mostrar "No hay registro de las historias de este
  Sprint" en lugar de las dos tablas, y la velocidad igual.
- **FR-009**: La velocidad MUST ser la que calcula HU-07 para el Sprint, sin recalcularla en la pantalla.
- **FR-010**: Un `{id}` inválido, inexistente, de otro proyecto o de un Sprint que no está "Finalizado" MUST responder
  la página 404 de HU-20a, decidida antes de calcular la velocidad; un `{id}` inválido, sin consultar la base.
- **FR-011**: El Sprint y sus historias MUST leerse en una sola lectura consistente; ante cualquier falla de lectura o
  del cálculo de HU-07, la pantalla MUST mostrar la página de error de HU-20a (500, o 503 si la base no responde) y
  MUST NOT mostrar datos parciales.
- **FR-012**: Todos los Sprints e historias mostrados MUST pertenecer al proyecto activo.
- **FR-013**: Sin proyecto activo MUST aplicarse la redirección de HU-20b, con regreso a la dirección pedida (la lista
  o el detalle).
- **FR-014**: Las pantallas MUST NOT tener formularios, acciones de modificación ni JavaScript, y la consulta MUST NOT
  modificar Sprints, historias ni el registro de cierre ni agregar migraciones.
- **FR-015**: Los textos de la pantalla MUST definirse en un único lugar; las respuestas de error MUST NOT incluir
  texto de consultas ni detalles internos.

## Entidades clave

- **Sprint** (HU-04, HU-15): número, objetivo, fechas y estado ("Activo" o "Finalizado"), del proyecto.
- **Registro de cierre** (HU-15): por cada historia que tenía el Sprint al cerrar, su estado y sus Story Points en ese
  momento. No cambia después del cierre.
- **Historia** (HU-02): título actual.
- **Métricas del Sprint** (HU-07): velocidad del Sprint finalizado.
- **Sprint anterior** (nuevo, no se guarda): un Sprint finalizado con sus historias completadas y no completadas.

## Criterios de éxito

- **SC-001**: En el 100% de las pruebas, el orden de la lista, las historias de cada tabla, sus Story Points y la
  velocidad coinciden con las cuentas a mano de "Datos de ejemplo".
- **SC-002**: En el 100% de los Sprints con registro de cierre de las pruebas, la velocidad mostrada es igual a la suma
  de los Story Points de las completadas del registro.
- **SC-003**: Ningún Sprint ni historia de otro proyecto aparece, y ninguna falla de lectura muestra datos parciales.
- **SC-004**: El Scrum Master llega al detalle de un Sprint en dos acciones (abrir la sección y elegir el Sprint), y
  cada pantalla responde en menos de 500 ms con la base local, 50 Sprints finalizados y un Sprint con 200 historias en
  el registro.
- **SC-005**: Cada bloque tiene pruebas que fallaron antes del código (evidencia RED) y pasan después (evidencia
  GREEN), y la cobertura de los archivos nuevos de `src/web` queda en `cobertura.txt`.

## Supuestos y decisiones

- "Planificación de proyecto.pdf" no está en el repositorio; se usaron el Issue (con sus datos de ejemplo), las notas
  del equipo y los datos de HU-09. Si al revisarlo aparece un conflicto, prevalece la planificación y se corrige esta
  especificación.
- HU-15 todavía no está en el código. Su registro de cierre, la tabla `sprint_story_results` con `sprint_id`,
  `story_id`, `final_status` y `story_points`, sale de `specs/015-sprint-close/data-model.md` (T001: HU-15). HU-16 se
  implementa cuando HU-15 esté en `main`.
- HU-07 calcula la velocidad con las historias "Hecho" que siguen asignadas al Sprint, con sus Story Points actuales.
  Después de un cierre de HU-15 son las mismas que las completadas del registro, porque HU-15 solo quita las no
  completadas y HU-13 y HU-17 bloquean la edición de las historias de un Sprint finalizado. La prueba de coherencia
  (SC-002) lo comprueba.
- Las fechas de los Sprints, las de los proyectos y el orden de creación de las historias son los de HU-09; el Issue de
  HU-16 no da las fechas del Sprint 3 ni las de los proyectos.
- El escenario 5 y el Sprint sin registro (nota 5 del equipo) usan un proyecto propio, «Gamma», porque el Issue no da
  datos para ellos.
- Las horas y los defectos de HU-09 no se cargan: no cambian la velocidad ni se muestran.
- El título de la historia es el actual (la base no guarda el título al cerrar).

## Trazabilidad

| Escenario | Requisitos | Éxito | Prueba de pantalla / BDD | Prueba de integración |
|---|---|---|---|---|
| 1 Listado | FR-002 | SC-001 | `TestHU16_Listado`, `TestPastSprintView`, `TestBDD_HU16_ListadoDeSprintsFinalizados` | `TestPostgresPastSprintCatalog` |
| 2 Sprint activo | FR-003 | SC-001 | `TestHU16_Listado` | `TestPostgresPastSprintCatalog` |
| 3 Detalle | FR-005, FR-006, FR-009 | SC-001, SC-002 | `TestHU16_Detalle`, `TestPastSprintDetailView`, `TestBDD_HU16_DetalleDeUnSprint` | `TestPostgresPastSprintCatalog`, `TestHU16_VelocidadCoherente` |
| 4 Todas completadas | FR-006, FR-007 | SC-001, SC-002 | `TestHU16_TodasCompletadas`, `TestPastSprintDetailView` | `TestHU16_VelocidadCoherente` |
| 5 Ninguna completada | FR-006, FR-007 | SC-001, SC-002 | `TestHU16_NingunaCompletada`, `TestPastSprintDetailView` | `TestPostgresPastSprintCatalog`, `TestHU16_VelocidadCoherente` |
| (Nota 5) Sprint sin registro | FR-008, FR-009 | SC-001 | `TestHU16_SinRegistro`, `TestPastSprintDetailView` | `TestPostgresPastSprintCatalog` |
| 6 Sin finalizados | FR-004 | — | `TestHU16_SinSprintsAnteriores`, `TestBDD_HU16_SinSprintsFinalizados` | `TestPostgresPastSprintCatalog` |
| 7 Solo lectura | FR-001, FR-014 | — | `TestHU16_SoloLectura`, `TestBDD_HU16_SoloLectura` | — |
| 8 Inexistente, en curso u otro proyecto | FR-010, FR-012 | SC-003 | `TestHU16_DetalleNoEncontrado` | `TestPostgresPastSprintCatalog` |
| 9 Error al consultar | FR-011, FR-015 | SC-003 | `TestHU16_ErrorAlConsultar` | — |
| (HU-20b) Sin proyecto | FR-013 | — | `TestHU16_SinProyecto`, `TestHU16_SinProyectoDetalle` | — |
| Rendimiento | — | SC-004 | `TestHU16_Rendimiento` | — |
| (Caso límite) Textos escapados | FR-002, FR-006 | — | `TestHU16_TextosEscapados` | — |
| (Registro) Dependencias nulas | FR-011 | — | `TestRegisterPastSprints_ArgumentosNil` | — |
| Sin migración ni cambios | FR-014 | — | — | revisión en T024 (`git diff --stat main`: solo archivos nuevos, `main.go` y `specs/016-past-sprints`) |
