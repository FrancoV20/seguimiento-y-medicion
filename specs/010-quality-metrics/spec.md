# Especificación: HU-10 – Cálculo de Métricas de Calidad y Avance

**Rama**: `HU-10`
**Creada**: 2026-10-28
**Estado**: Borrador
**Constitución**: v1.1.0
**Depende de**: HU-06, HU-07, HU-14, HU-15, HU-19, HU-20a, HU-20b (HU-19 solo para tener defectos resueltos reales en
la demo; el código no la usa)
**Orden de implementación**: después de HU-15 (crea `sprint_story_results`) y de HU-06 (crea `defects`); los dos
tienen que estar en `main` antes de crear la rama (T001)
**Habilita**: HU-08 (gráficos) y HU-09 (PDF), que consumen el cálculo
**Prioridad**: Media (Should Have) · **Valor**: 75 · **Estimación**: 5 SP

**Entrada**: "HU-10 / Cálculo de Métricas de Calidad y Avance. Como Scrum Master, quiero que el sistema calcule el
porcentaje de historias completadas, los Story Points planificados y completados, y la cantidad de defectos
detectados y resueltos por Sprint, y verlos en una pantalla, para evaluar la calidad y el avance real del proyecto
más allá del esfuerzo invertido."

## Historia de usuario

Como Scrum Master, quiero que el sistema calcule el porcentaje de historias completadas, los Story Points
planificados y completados, y la cantidad de defectos detectados y resueltos por Sprint, y verlos en una pantalla,
para evaluar la calidad y el avance real del proyecto más allá del esfuerzo invertido.

> **Alcance**: HU-10 calcula, para un Sprint finalizado del proyecto activo, el avance (historias planificadas,
> completadas y porcentaje), los Story Points planificados y completados, y los defectos detectados y resueltos. Lo
> muestra en una pantalla propia, "Calidad y avance", de solo lectura. No guarda nada: calcula al abrir.
>
> **Por qué después de HU-15**: el avance sale del registro que HU-15 guarda al cerrar el Sprint. Sin ese registro el
> avance siempre daría 100 %, porque el cierre quita de las asignaciones las historias no completadas.
>
> **Relación con HU-25**: HU-25 ("Métricas del Sprint") deja afuera estas métricas. HU-10 tiene su propia pantalla y
> no toca la de HU-25.

## Clarifications

### Session 2026-10-28

Decisiones del equipo registradas antes de `/speckit-clarify` (no se vuelven a preguntar):

- Q: ¿De dónde salen las historias planificadas y completadas? → A: Del registro de cierre de HU-15: planificadas
  = todas las filas del Sprint; completadas = las de estado final "Hecho".
- Q: ¿Cómo se calculan los Story Points? → A: Planificados = suma de los Story Points informados de todas las
  historias del registro; completados = suma de los de las "Hecho". Las historias sin Story Points no suman y se
  avisa cuántas son, con singular o plural correcto.
- Q: ¿Cómo se calcula y muestra el porcentaje? → A: Completadas ÷ planificadas × 100, redondeado mitad hacia arriba
  a dos decimales con la misma regla que HU-07, con coma: "75,00 %". Con 0 planificadas: "No disponible".
- Q: ¿Qué defectos cuentan? → A: Detectados = los del proyecto cuyo Sprint de detección es el Sprint. Resueltos =
  los del proyecto en estado "Resuelto" cuyo Sprint de resolución es el Sprint, aunque se hayan detectado antes. Son
  dos conteos independientes.
- Q: ¿Se guarda el resultado? → A: No. Se calcula al abrir, sin migración, porque los defectos se pueden reabrir
  (HU-19) y un valor guardado quedaría desactualizado.
- Q: ¿Qué Sprints se calculan? → A: Solo los "Finalizado" del proyecto activo. Uno activo se rechaza con el error de
  HU-07 de Sprint no finalizado; uno inexistente o de otro proyecto, con un error propio de "Sprint no encontrado".
- Q: ¿Cómo se elige el Sprint en la pantalla? → A: Lista de Sprints finalizados del más reciente al más antiguo; por
  defecto el más reciente. Un Sprint inválido, inexistente, activo o de otro proyecto en la dirección se ignora,
  como en HU-25.

## Sección SDD

### Intención

Dar al Scrum Master una medida del avance real y de la calidad de cada Sprint cerrado: cuánto de lo planificado se
completó (en historias y en Story Points) y cuántos defectos aparecieron y se resolvieron, para complementar la
velocidad y el esfuerzo de HU-07.

### Alcance

- Cálculo de calidad y avance de un Sprint finalizado del proyecto activo.
- Pantalla "Calidad y avance" con la lista de Sprints finalizados y los números del elegido.
- Aviso de historias sin Story Points.
- Aviso de que el Sprint activo se calcula al cerrarlo.
- Estados vacíos: sin Sprints finalizados, Sprint sin historias registradas, Sprint sin defectos.
- Un cálculo reutilizable por HU-08 y HU-09.

### Fuera de alcance

- Gráficos (HU-08), exportar a PDF (HU-09) y detalle de las historias del Sprint (HU-16).
- Defectos abiertos del proyecto y otros indicadores de defectos.
- Porcentaje de avance por Story Points.
- Guardar un histórico de estas métricas.
- Las métricas de HU-25 y de HU-07 (velocidad, horas, desviación): no se cambian.
- Calcular métricas de un Sprint activo.

### Entradas

| Origen | Dato | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `GET /calidad` | `sprint` (en la dirección) | identificador numérico | no | Sprint finalizado del proyecto activo; ausente o inválido = el más reciente |
| Contexto (HU-20b) | proyecto activo | identificador | sí | Nunca sale de la dirección ni de un formulario |

### Salidas esperadas

- Con Sprints finalizados: 200 con la lista de Sprints (del más reciente al más antiguo) y, para el elegido:
  - "X de Y historias completadas" y el porcentaje ("75,00 %" o "No disponible");
  - Story Points planificados y completados;
  - la advertencia de historias sin Story Points, si las hay;
  - defectos detectados y resueltos.
- Si además hay un Sprint activo: el aviso "El Sprint N sigue en curso; sus métricas se calculan al cerrarlo".
- Sin Sprints finalizados: 200 con "Aún no hay Sprints finalizados" (y el aviso del Sprint activo, si hay uno).
- Falla al leer: la página de error de HU-20a (500 o 503), sin números parciales.
- Sin proyecto activo: redirección 303 de HU-20b a elegir proyecto, con regreso a "Calidad y avance".
- Cualquier método distinto de `GET`: 405 de HU-20a.

### Reglas de negocio

- R1: solo se calculan Sprints "Finalizado" del proyecto activo.
- R2: historias planificadas = las del registro de cierre del Sprint (HU-15); completadas = las de estado final
  "Hecho".
- R3: porcentaje = completadas ÷ planificadas × 100, redondeado mitad hacia arriba a dos decimales (regla de HU-07),
  con coma decimal y " %". Sin historias planificadas: "No disponible".
- R4: Story Points planificados = suma de los Story Points de las historias del registro que los tienen; completados
  = suma de los de las "Hecho". Son los Story Points que la historia tenía al cerrar.
- R5: si hay historias sin Story Points, se avisa: "1 historia sin Story Points no se suma a los planificados" o "N
  historias sin Story Points no se suman a los planificados".
- R6: los Story Points completados coinciden con la velocidad del mismo Sprint calculada por HU-07.
- R7: defectos detectados = defectos del proyecto cuyo Sprint de detección es el Sprint.
- R8: defectos resueltos = defectos del proyecto en estado "Resuelto" cuyo Sprint de resolución es el Sprint. Un
  defecto reabierto deja de contar como resuelto y sigue contando como detectado en su Sprint de detección.
- R9: nada se guarda; cada apertura calcula con los datos del momento.
- R10: todos los datos de un Sprint se leen juntos, de modo que los números sean coherentes entre sí.

### Restricciones

- Sin migración ni tablas nuevas; sin dependencias nuevas; sin JavaScript.
- No se editan archivos de HU-07 ni de HU-25. El único archivo compartido que se edita es el arranque del servidor
  (se suma un bloque de registro).
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- Los textos de la pantalla se definen en un único lugar.

### Condiciones de error

| Condición | Resultado | Respuesta |
|---|---|---|
| Sin Sprints finalizados | "Aún no hay Sprints finalizados" | 200 |
| `sprint` inválido, inexistente, activo o de otro proyecto | Se ignora y se muestra el más reciente | 200 |
| Sprint sin historias en el registro de cierre | Porcentaje "No disponible" | 200 |
| Falla al leer la lista o los datos del Sprint | Página de error de HU-20a, sin números parciales | 500 o 503 |
| Sin proyecto activo | Mensaje de HU-20b | 303 |
| Método distinto de `GET` | Página de HU-20a | 405 |
| Cálculo pedido para un Sprint activo (desde otra historia) | Error de Sprint no finalizado (HU-07) | — |
| Cálculo pedido para un Sprint inexistente o de otro proyecto | Error de Sprint no encontrado | — |

## Escenarios BDD

El escenario 10 del Issue combina tres comprobaciones; se escribe como 10a, 10b y 10c, sin "Esquema del escenario".

### Bloque A – Avance

**Escenario 1: Porcentaje de avance**
- **Dado** un Sprint finalizado que tenía 4 historias y cerró con 3 en "Hecho" (la otra volvió al Backlog)
- **Cuando** abro la calidad y el avance de ese Sprint
- **Entonces** veo 3 de 4 historias completadas y "75,00 %"

**Escenario 2: Ninguna historia completada**
- **Dado** un Sprint finalizado cuyas historias no llegaron a "Hecho"
- **Cuando** abro su calidad y avance
- **Entonces** veo 0 historias completadas y "0,00 %", sin error

**Escenario 3: Sprint sin historias registradas**
- **Dado** un Sprint finalizado sin historias en su registro de cierre
- **Cuando** abro su calidad y avance
- **Entonces** veo el porcentaje como "No disponible" en lugar de un error o un valor inválido

### Bloque B – Story Points

**Escenario 4: Story Points planificados y completados**
- **Dado** un Sprint finalizado con historias de 3, 5, 8 y 2 Story Points, de las cuales las de 3 y 5 quedaron en
  "Hecho"
- **Cuando** abro su calidad y avance
- **Entonces** veo 18 Story Points planificados y 8 completados, y los 8 coinciden con la velocidad del Sprint
  (HU-07)

**Escenario 5: Historias sin Story Points**
- **Dado** un Sprint finalizado con una historia sin Story Points
- **Cuando** abro su calidad y avance
- **Entonces** los planificados suman solo las historias estimadas y veo la advertencia "1 historia sin Story Points
  no se suma a los planificados"

### Bloque C – Defectos

**Escenario 6: Defectos detectados y resueltos en el Sprint**
- **Dado** el Sprint 2 finalizado con 3 defectos detectados en él, uno de ellos resuelto en él, y un defecto detectado
  en el Sprint 1 y resuelto en el Sprint 2
- **Cuando** abro su calidad y avance
- **Entonces** veo 3 defectos detectados y 2 resueltos

**Escenario 7: Sprint sin defectos**
- **Dado** un Sprint finalizado sin defectos detectados ni resueltos
- **Cuando** abro su calidad y avance
- **Entonces** veo 0 defectos detectados y 0 resueltos, sin error

**Escenario 8: Defecto reabierto**
- **Dado** un defecto que se resolvió en un Sprint finalizado y luego se reabrió
- **Cuando** abro la calidad y el avance de ese Sprint
- **Entonces** el defecto sigue contando como detectado en su Sprint de detección y ya no cuenta como resuelto

**Escenario 9: Datos de otro proyecto**
- **Dado** dos proyectos con Sprints, historias y defectos propios
- **Cuando** abro la calidad y el avance de un Sprint del proyecto activo
- **Entonces** solo cuento las historias y los defectos de ese proyecto

### Bloque D – Pantalla

**Escenario 10a: Lista de Sprints y Sprint por defecto**
- **Dado** un proyecto con los Sprints 1, 2 y 3 finalizados
- **Cuando** abro "Calidad y avance"
- **Entonces** veo la lista "Sprint 3", "Sprint 2", "Sprint 1", en ese orden, y los datos del Sprint 3

**Escenario 10b: Elegir otro Sprint**
- **Dado** la pantalla "Calidad y avance" con los Sprints 1, 2 y 3 finalizados
- **Cuando** elijo el Sprint 1
- **Entonces** veo los datos del Sprint 1

**Escenario 10c: El Sprint activo no figura**
- **Dado** un proyecto con Sprints finalizados y el Sprint 4 activo
- **Cuando** abro "Calidad y avance"
- **Entonces** el Sprint 4 no figura en la lista y veo "El Sprint 4 sigue en curso; sus métricas se calculan al
  cerrarlo"

**Escenario 11: Sin Sprints finalizados**
- **Dado** un proyecto sin Sprints finalizados
- **Cuando** abro "Calidad y avance"
- **Entonces** veo "Aún no hay Sprints finalizados" y no veo un error ni una pantalla vacía

**Escenario 12: Error al leer los datos**
- **Dado** un Sprint finalizado y un fallo al leer sus datos
- **Cuando** abro su calidad y avance
- **Entonces** veo la página de error de la aplicación y no veo números parciales

### Casos límite

- Un `sprint` vacío, no numérico, negativo, cero, con ceros a la izquierda o mayor que el máximo se trata como
  ausente: se muestra el Sprint finalizado más reciente, sin mensaje de error.
- Un `sprint` de un Sprint activo, inexistente o de otro proyecto también se ignora; nunca se muestran datos de otro
  proyecto.
- Un Sprint finalizado sin defectos y sin historias muestra "No disponible", 0 Story Points y 0 defectos.
- Todas las historias sin Story Points: planificados 0, completados 0 y la advertencia con el total, en plural si son
  varias.
- Una historia "Hecho" sin Story Points en el registro (por datos cargados a mano; HU-07 no deja cerrar así) cuenta
  como completada y en la advertencia, y no suma Story Points.
- Un defecto detectado y resuelto en el mismo Sprint cuenta una vez en cada conteo.
- Un defecto detectado en el Sprint 2 y resuelto en el Sprint 3 cuenta como detectado en el 2 y resuelto en el 3.
- Un defecto de otro proyecto nunca cuenta, aunque apunte a un Sprint del proyecto activo.
- El porcentaje se redondea mitad hacia arriba: 2 de 3 → "66,67 %"; 1 de 8 → "12,50 %"; 1 de 3 → "33,33 %".
- Cambios posteriores al cierre en una historia (por ejemplo, sus Story Points) no cambian los números del Sprint:
  se usan los del registro de cierre.
- Un Sprint "Finalizado" cerrado sin el registro de HU-15 (datos anteriores o cargados a mano) muestra
  "No disponible" y 0 Story Points; los defectos se cuentan igual.
- Recargar la pantalla vuelve a calcular con los datos actuales: si un defecto se reabrió, los resueltos bajan.
- Los números de Sprint se muestran escapados y la pantalla no muestra textos ingresados por usuarios.

## Requisitos funcionales

- **FR-001**: El sistema MUST ofrecer la sección "Calidad y avance" en `/calidad`, que exige proyecto activo, y
  MUST responder solo a `GET` (y `HEAD`, que el enrutador de HU-20a atiende igual); cualquier otro método responde
  405.
- **FR-002**: La pantalla MUST listar los Sprints finalizados del proyecto activo, del más reciente al más antiguo,
  y mostrar por defecto los datos del más reciente.
- **FR-003**: Elegir un Sprint de la lista MUST mostrar sus datos; un `sprint` inválido, inexistente, activo o de
  otro proyecto MUST ignorarse y mostrar el más reciente.
- **FR-004**: El Sprint activo MUST NOT figurar en la lista; si existe, la pantalla MUST mostrar "El Sprint N sigue
  en curso; sus métricas se calculan al cerrarlo".
- **FR-005**: Sin Sprints finalizados, la pantalla MUST mostrar "Aún no hay Sprints finalizados", con respuesta 200.
- **FR-006**: El sistema MUST calcular las historias planificadas (todas las del registro de cierre del Sprint) y las
  completadas (las de estado final "Hecho").
- **FR-007**: El sistema MUST calcular el porcentaje de historias completadas con la regla R3 y mostrarlo con coma
  y " %"; sin historias planificadas MUST mostrar "No disponible".
- **FR-008**: El sistema MUST calcular los Story Points planificados y completados con la regla R4, usando los
  Story Points del registro de cierre.
- **FR-009**: Si hay historias sin Story Points, la pantalla MUST mostrar la advertencia de R5 con singular o plural
  correcto; si no hay, MUST NOT mostrarla.
- **FR-010**: Los Story Points completados MUST coincidir con la velocidad que calcula HU-07 para el mismo Sprint.
- **FR-011**: El sistema MUST contar los defectos detectados (R7) y resueltos (R8) del Sprint como dos conteos
  independientes, solo con defectos del proyecto activo.
- **FR-012**: Un defecto reabierto MUST dejar de contar como resuelto y MUST seguir contando como detectado.
- **FR-013**: Todas las historias y defectos contados MUST pertenecer al proyecto activo.
- **FR-014**: El cálculo MUST hacerse solo para Sprints "Finalizado" del proyecto pedido: un Sprint activo MUST
  devolver el error de Sprint no finalizado de HU-07, y uno inexistente o de otro proyecto, el error de Sprint no
  encontrado.
- **FR-015**: Los datos de un Sprint MUST leerse en una sola lectura consistente; ante cualquier falla, la pantalla
  MUST mostrar la página de error de HU-20a y MUST NOT mostrar números parciales.
- **FR-016**: El sistema MUST NOT guardar los resultados ni agregar migraciones.
- **FR-017**: Sin proyecto activo MUST aplicarse la redirección de HU-20b, con regreso a "Calidad y avance".
- **FR-018**: El cálculo MUST quedar disponible para otras historias (HU-08, HU-09) con el proyecto y el Sprint como
  datos de entrada.
- **FR-019**: Los textos de la pantalla MUST definirse en un único lugar; las respuestas de error MUST NOT incluir
  texto de consultas ni detalles internos.

## Entidades clave

- **Sprint** (HU-04): número, estado ("Activo" o "Finalizado") y proyecto.
- **Registro de cierre** (HU-15): por cada historia que tenía el Sprint al cerrar, su estado final y sus Story
  Points en ese momento.
- **Defecto** (HU-06): proyecto, Sprint de detección, estado ("Abierto" o "Resuelto") y Sprint de resolución
  (HU-19).
- **Calidad y avance del Sprint** (nuevo, no se guarda): historias planificadas y completadas, porcentaje, Story
  Points planificados y completados, historias sin Story Points, defectos detectados y resueltos.

## Criterios de éxito

- **SC-001**: En el 100% de las pruebas, el porcentaje, los Story Points y los conteos de defectos coinciden con el
  cálculo a mano sobre los datos de prueba, incluidos los redondeos de los casos límite.
- **SC-002**: En el 100% de las pruebas con un Sprint cerrado, los Story Points completados coinciden con la
  velocidad de HU-07.
- **SC-003**: Ningún dato de otro proyecto ni de un Sprint activo aparece en la lista ni en los números.
- **SC-004**: Ninguna falla de lectura muestra números parciales.
- **SC-005**: El Scrum Master ve los números de un Sprint en una acción (abrir la sección) o dos (elegir otro
  Sprint), y la pantalla responde en menos de 500 ms con la base local, 20 Sprints finalizados, 50 historias por
  Sprint y 100 defectos.
- **SC-006**: Cada bloque tiene pruebas que fallaron antes del código (evidencia RED) y pasan después (evidencia
  GREEN), y la cobertura de `src/metrics` queda en `cobertura.txt` con al menos 80%.

## Supuestos y decisiones

- Los nombres de tablas y columnas de HU-06 (`defects`) y HU-15 (`sprint_story_results`) salen de sus documentos
  (`specs/006-defects`, `specs/015-sprint-close`), porque todavía no están en el código de `main`; se confirman en
  T001.
- HU-07 ya no deja cerrar un Sprint con historias "Hecho" sin Story Points (validación de cierre); por eso, en la
  práctica, las historias sin Story Points del registro son las no completadas.
- La velocidad de HU-07 se calcula con las historias "Hecho" que quedan asignadas al Sprint y sus Story Points
  actuales; coincide con los del registro de cierre mientras esos Story Points no cambien después del cierre. Hoy
  ninguna pantalla de `main` edita Story Points; cuando llegue HU-13, bloquea la edición de historias de un Sprint
  finalizado.
- HU-19 hace falta para tener defectos resueltos reales en la demo; las pruebas cargan los datos en la base.
- "Más reciente" es el de mayor número de Sprint.
- La planificación del proyecto (PDF) lista como dependencias HU-14, HU-15 y HU-19; el Issue agrega HU-06, HU-07,
  HU-20a y HU-20b. Se toman todas.

## Trazabilidad

| Escenario | Requisitos | Éxito | Prueba de pantalla / BDD | Prueba de dominio / integración |
|---|---|---|---|---|
| 1 Porcentaje de avance | FR-006, FR-007 | SC-001 | `TestHU10_NumerosDelSprint`, `TestQualityView`, `TestWebQualityBDD_PorcentajeDeAvance` | `TestCalculateSprintQuality`, `TestPostgresQuality_Historias` |
| 2 Ninguna completada | FR-006, FR-007 | SC-001 | `TestHU10_ElegirOtroSprint` | `TestCalculateSprintQuality`, `TestPostgresQuality_Historias` |
| 3 Sin historias registradas | FR-007 | SC-001 | `TestHU10_NumerosDelSprint` | `TestCalculateSprintQuality`, `TestPostgresQuality_Historias` |
| 4 Story Points planificados y completados | FR-008, FR-010 | SC-001, SC-002 | `TestHU10_NumerosDelSprint` | `TestCalculateSprintQuality`, `TestPostgresQuality_Historias`, `TestQuality_CoherenciaConVelocidadHU07` |
| 5 Historias sin Story Points | FR-008, FR-009 | SC-001 | `TestHU10_NumerosDelSprint`, `TestQualityView` | `TestCalculateSprintQuality`, `TestPostgresQuality_Historias` |
| 6 Defectos detectados y resueltos | FR-011 | SC-001 | `TestHU10_NumerosDelSprint`, `TestWebQualityBDD_DefectosDetectadosYResueltos` | `TestCalculateSprintQuality`, `TestPostgresQuality_Defectos` |
| 7 Sprint sin defectos | FR-011 | SC-001 | `TestHU10_ListaYSprintPorDefecto`, `TestHU10_ElegirOtroSprint` | `TestPostgresQuality_Defectos` |
| 8 Defecto reabierto | FR-012 | SC-001 | — | `TestPostgresQuality_Defectos` |
| 9 Datos de otro proyecto | FR-013 | SC-003 | `TestHU10_OtroProyecto` | `TestPostgresQuality_OtroProyecto`, `TestPostgresQualityCatalog` |
| 10a Lista y Sprint por defecto | FR-002 | SC-005 | `TestHU10_ListaYSprintPorDefecto`, `TestWebQualityBDD_ListaDeSprintsYSprintPorDefecto` | `TestPostgresQualityCatalog` |
| 10b Elegir otro Sprint | FR-003 | SC-005 | `TestHU10_ElegirOtroSprint`, `TestHU10_SprintInvalidoSeIgnora`, `TestQualityView` | — |
| 10c Sprint activo no figura | FR-004 | SC-003 | `TestHU10_SprintActivoNoFigura` | `TestPostgresQualityCatalog`, `TestPostgresQuality_SprintActivoEInexistente` |
| 11 Sin Sprints finalizados | FR-005 | — | `TestHU10_SinSprintsFinalizados`, `TestWebQualityBDD_SinSprintsFinalizados` | `TestPostgresQualityCatalog` |
| 12 Error al leer | FR-015, FR-019 | SC-004 | `TestHU10_ErrorAlLeer` | `TestQualityService_SprintQuality`, `TestPostgresQualityCatalog` |
| (HU-20b) Sin proyecto | FR-017 | — | `TestHU10_SinProyecto` | — |
| (HU-20a) Solo lectura | FR-001 | — | `TestHU10_PostNoPermitido` | — |
| Rendimiento | — | SC-005 | `TestHU10_Rendimiento` | — |
| Cálculo reutilizable y sin persistencia | FR-014, FR-016, FR-018 | — | — | `TestQualityService_SprintQuality`, `TestPostgresQuality_SprintActivoEInexistente` |

Nombres confirmados contra el código en T037. Las pruebas de pantalla están en `src/web/quality_handlers_test.go`,
`src/web/quality_view_test.go` y `src/web/postgres_quality_catalog_integration_test.go`; las BDD en
`test/web_quality_bdd_test.go`; las de dominio e integración en `src/metrics/quality_test.go`,
`src/metrics/quality_service_test.go` y `src/metrics/postgres_quality_store_integration_test.go`. La verificación
manual de los escenarios con la aplicación en ejecución está en `evidencia-manual.txt` (T036).
