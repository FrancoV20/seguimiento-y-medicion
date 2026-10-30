# Especificación: HU-12 – Consulta del Estado de un Proyecto

**Rama**: `HU-12`
**Creada**: 2026-10-28
**Estado**: Borrador
**Constitución**: v1.1.0
**Depende de**: HU-01, HU-14, HU-20a, HU-20b (verificación manual: HU-23 para tener un Sprint en curso)
**Habilita**: HU-09 (reporte del proyecto en PDF), que reutiliza la regla de estado y la de avance
**Prioridad**: Media (Should Have) · **Valor**: 75 · **Estimación**: 3 SP (sugerida)

**Entrada**: "HU-12 / Consulta del Estado de un Proyecto. Como Scrum Master, quiero consultar el estado de un
proyecto (no iniciado, en curso o finalizado) junto con su avance, para saber en qué etapa se encuentra."

## Historia de usuario

Como Scrum Master, quiero consultar el estado de un proyecto (no iniciado, en curso o finalizado) junto con su
avance, para saber en qué etapa se encuentra.

> **Alcance**: HU-12 agrega la sección "Estado del proyecto", de solo lectura, con cuatro datos del proyecto activo:
> su estado según las fechas, el avance ("X de Y historias completadas (P %)"), las historias por estado y el Sprint
> en curso. No guarda nada: calcula al abrir.
>
> **No recalcula lo que ya existe**: el Sprint en curso sale del Tablero (HU-14). Lo nuevo del dominio son dos reglas
> puras, el estado por fechas y el porcentaje de avance, que HU-09 reutiliza en el reporte del proyecto.
>
> **Por qué una sección nueva**: la lista de "Proyectos" la edita HU-11; HU-12 no toca esos archivos.

## Clarifications

### Session 2026-10-28

Decisiones del equipo registradas antes de `/speckit-clarify` (no se vuelven a preguntar):

- Q: ¿Cómo se ofrece la pantalla? → A: Sección "Estado del proyecto" en `/estado`, que exige proyecto activo, de
  solo lectura (cualquier otro método → 405 de HU-20a), sin JavaScript, sin migración y sin guardar nada. No se
  editan archivos de HU-21 ni de HU-11: la sección es nueva.
- Q: ¿Cómo se calcula el estado? → A: Con las fechas del proyecto y la fecha de hoy: hoy anterior al inicio → "No
  iniciado"; hoy entre el inicio y el fin, los dos días incluidos → "En curso"; hoy posterior al fin → "Finalizado".
  Se comparan fechas, no horas. "Hoy" es la fecha local del servidor. No se usa el estado guardado del proyecto, que
  tiene un valor fijo.
- Q: ¿Cómo se calcula el avance? → A: Historias "Hecho" del proyecto activo ÷ total de historias del proyecto × 100,
  redondeado mitad hacia arriba a dos decimales (la misma regla que HU-07), con coma: "X de Y historias completadas
  (P %)". Sin historias: "0 de 0 historias completadas (0,00 %)".
- Q: ¿Qué historias por estado se muestran? → A: Pendiente, En Sprint, En progreso y Hecho, en ese orden, incluidos
  los ceros. Se leen junto con el nombre y las fechas del proyecto, en una sola lectura consistente.
- Q: ¿De dónde sale el Sprint en curso? → A: Del Tablero de HU-14, sin modificarlo: "Sprint N en curso", su objetivo
  y sus fechas (dd/mm/aaaa). Sin Sprint activo: "No hay un Sprint en curso".
- Q: ¿Qué pasa si falla una lectura? → A: La página de error de HU-20a (500, o 503 si la base no responde), sin
  datos parciales. Sin proyecto activo, la redirección de HU-20b con regreso a "Estado del proyecto".
- Q: ¿Qué contrato usa HU-09? → A: Dos reglas puras: la de estado recibe inicio, fin y la fecha de hoy (no lee el
  reloj; el reloj solo lo usa la pantalla), y la de avance recibe las historias "Hecho" y el total y devuelve el
  porcentaje como texto con punto y dos decimales ("57.14", "0.00"). La pantalla lo muestra con coma ("57,14 %").
- Q: ¿Qué queda afuera? → A: Días restantes, el estado de todos los proyectos en la lista de "Proyectos" (choca con
  HU-11) y las métricas por Sprint (HU-10, HU-25).

## Sección SDD

### Intención

Que el Scrum Master sepa de un vistazo en qué etapa está el proyecto activo (si todavía no empezó, si está en curso o
si ya terminó), cuánto se avanzó en historias y qué Sprint está corriendo.

### Alcance

- Sección "Estado del proyecto" del proyecto activo, de solo lectura.
- Estado por fechas: "No iniciado", "En curso" o "Finalizado".
- Avance: "X de Y historias completadas (P %)".
- Historias por estado: Pendiente, En Sprint, En progreso y Hecho, con ceros.
- Sprint en curso: número, objetivo y fechas, o "No hay un Sprint en curso".
- Dos reglas del dominio reutilizables por HU-09.

### Fuera de alcance

- Días restantes o transcurridos.
- El estado de todos los proyectos en la lista de "Proyectos" (HU-11 edita esa pantalla).
- Métricas por Sprint (HU-10, HU-25) y gráficos (HU-08).
- Cambiar el estado guardado del proyecto o las fechas (HU-11).

### Entradas

| Origen | Dato | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `GET /estado` | — | — | — | La pantalla no recibe parámetros |
| Contexto (HU-20b) | proyecto activo | identificador | sí | Nunca sale de la dirección ni de un formulario |
| Reloj del servidor | fecha de hoy | fecha local | sí | En las pruebas, 28/10/2026 |

### Salidas esperadas

- 200 con el nombre y las fechas del proyecto, el estado, el avance, las historias por estado y el bloque del Sprint
  en curso (o "No hay un Sprint en curso").
- Falla al leer: la página de error de HU-20a (500 o 503), sin datos parciales.
- Proyecto activo que ya no existe (borrado entre la elección y la consulta): la página 404 de HU-20a.
- Sin proyecto activo: redirección 303 de HU-20b a elegir proyecto, con regreso a "Estado del proyecto".
- Cualquier método distinto de `GET`: 405 de HU-20a.

### Reglas de negocio

- R1: estado = "No iniciado" si hoy es anterior a la fecha de inicio; "En curso" si hoy está entre el inicio y el fin,
  los dos días incluidos; "Finalizado" si hoy es posterior a la fecha de fin. Se comparan días, no horas.
- R2: "hoy" es la fecha local del servidor en el momento de la consulta.
- R3: avance = historias "Hecho" del proyecto ÷ total de historias del proyecto × 100, redondeado mitad hacia arriba
  a dos decimales (regla de HU-07). Se muestra "X de Y historias completadas (P %)" con coma decimal. Sin historias:
  "0 de 0 historias completadas (0,00 %)", sin dividir por cero.
- R4: historias por estado = todas las historias del proyecto activo, en su estado actual, contadas en Pendiente, En
  Sprint, En progreso y Hecho, en ese orden; un estado sin historias muestra 0. La suma es el total del avance.
- R5: Sprint en curso = el Sprint activo del proyecto que muestra el Tablero (HU-14): "Sprint N en curso", su
  objetivo y "del dd/mm/aaaa al dd/mm/aaaa". Sin Sprint activo: "No hay un Sprint en curso".
- R6: el nombre, las fechas y las cantidades se leen juntos, de modo que sean coherentes entre sí.
- R7: nada se guarda; cada apertura calcula con los datos y la fecha del momento.

### Restricciones

- Sin migración ni tablas nuevas; sin dependencias nuevas; sin JavaScript.
- No se editan archivos de HU-14, HU-21 ni HU-11 ni los existentes del dominio de proyectos. El único archivo
  compartido que se edita es el arranque del servidor (se suma un bloque de registro).
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- Los textos de la pantalla se definen en un único lugar.

### Condiciones de error

| Condición | Resultado | Respuesta |
|---|---|---|
| Proyecto sin historias | Cuatro cantidades en 0 y "0 de 0 historias completadas (0,00 %)" | 200 |
| Proyecto sin Sprint activo | "No hay un Sprint en curso" y el resto igual | 200 |
| Falla al leer el proyecto, las cantidades o el Tablero | Página de error de HU-20a, sin datos parciales | 500 o 503 |
| Proyecto activo que ya no existe | Página 404 de HU-20a | 404 |
| Sin proyecto activo | Mensaje de HU-20b | 303 |
| Método distinto de `GET` | Página de HU-20a | 405 |

## Escenarios BDD

Los datos de ejemplo son los del Issue; cada escenario usa su propio proyecto (ver "Datos de ejemplo").

### Bloque A – Estado por fechas

**Escenario 1: Proyecto en curso**
- **Dado** el proyecto «Alfa», cuya fecha de hoy está entre su inicio y su fin
- **Cuando** consulto su estado
- **Entonces** veo "En curso", "Sprint 2 en curso" con su objetivo y sus fechas, las historias por estado (Pendiente
  2, En Sprint 1, En progreso 1, Hecho 4) y "4 de 8 historias completadas (50,00 %)"

**Escenario 2: Proyecto no iniciado**
- **Dado** el proyecto «Beta», cuya fecha de inicio es futura
- **Cuando** consulto su estado
- **Entonces** veo "No iniciado"

**Escenario 3: Proyecto finalizado**
- **Dado** el proyecto «Gamma», cuya fecha de fin ya pasó
- **Cuando** consulto su estado
- **Entonces** veo "Finalizado" y "3 de 3 historias completadas (100,00 %)"

**Escenario 4: Proyecto que empieza hoy**
- **Dado** el proyecto «Delta», que empieza hoy
- **Cuando** consulto su estado
- **Entonces** veo "En curso"

**Escenario 5: Proyecto que termina hoy**
- **Dado** el proyecto «Épsilon», que termina hoy
- **Cuando** consulto su estado
- **Entonces** veo "En curso"

### Bloque B – Avance y Sprint

**Escenario 6: Proyecto sin historias**
- **Dado** el proyecto «Beta», sin historias
- **Cuando** consulto su estado
- **Entonces** veo las cuatro cantidades en 0 y "0 de 0 historias completadas (0,00 %)", sin error ni división por
  cero

**Escenario 7: Sin Sprint en curso**
- **Dado** el proyecto «Gamma», sin Sprint activo
- **Cuando** consulto su estado
- **Entonces** veo "No hay un Sprint en curso" y el resto de la información igual

**Escenario 8: Datos de otro proyecto**
- **Dado** que el proyecto activo es «Alfa»
- **Cuando** consulto su estado
- **Entonces** las cantidades y el Sprint son solo de «Alfa», sin contar las historias de «Gamma»

### Bloque C – Contexto y errores

**Escenario 9: Sin proyecto seleccionado**
- **Dado** que no elegí un proyecto
- **Cuando** abro "Estado del proyecto"
- **Entonces** se me pide elegir uno y, al elegirlo, vuelvo a "Estado del proyecto"

**Escenario 10: Error al consultar**
- **Dado** un proyecto existente
- **Cuando** ocurre un error al leer los datos
- **Entonces** veo la página de error de la aplicación, sin detalles internos y sin datos parciales

### Datos de ejemplo

Fecha de hoy en las pruebas: 28/10/2026.

| Proyecto | Inicio | Fin | Historias | Sprint en curso | Estado esperado |
|---|---|---|---|---|---|
| «Alfa» | 01/10/2026 | 30/11/2026 | Pendiente 2, En Sprint 1, En progreso 1, Hecho 4 (8) | Sprint 2 "Pagos online", del 19/10/2026 al 01/11/2026 | En curso |
| «Beta» | 01/12/2026 | 31/01/2027 | ninguna | ninguno (sin Sprints) | No iniciado |
| «Gamma» | 01/08/2026 | 30/09/2026 | Hecho 3 (3) | ninguno | Finalizado |
| «Delta» | 28/10/2026 | 15/12/2026 | ninguna | ninguno | En curso (empieza hoy) |
| «Épsilon» | 01/09/2026 | 28/10/2026 | ninguna | ninguno | En curso (termina hoy) |

Cuentas:

- «Alfa»: 2 + 1 + 1 + 4 = 8 historias; 4 ÷ 8 × 100 = 50 → "4 de 8 historias completadas (50,00 %)". 28/10/2026 está
  entre el 01/10/2026 y el 30/11/2026 → "En curso". "Sprint 2 en curso", "Pagos online", "del 19/10/2026 al
  01/11/2026".
- «Beta»: 28/10/2026 es anterior al 01/12/2026 → "No iniciado"; 0 historias → cantidades 0, 0, 0 y 0 y "0 de 0
  historias completadas (0,00 %)"; sin Sprints → "No hay un Sprint en curso".
- «Gamma»: 28/10/2026 es posterior al 30/09/2026 → "Finalizado"; 3 ÷ 3 × 100 = 100 → "3 de 3 historias completadas
  (100,00 %)"; cantidades 0, 0, 0 y 3; sin Sprint activo → "No hay un Sprint en curso".
- «Delta»: el inicio es hoy → "En curso". «Épsilon»: el fin es hoy → "En curso". Los dos sin historias: "0 de 0
  historias completadas (0,00 %)".
- Escenario 8: con «Alfa» activo se ven 8 historias y 4 "Hecho"; si se mezclaran las 3 de «Gamma», serían 11 y 7
  ("7 de 11", 63,64 %).

### Casos límite

- El redondeo es mitad hacia arriba: 2 de 3 → "66,67 %"; 1 de 3 → "33,33 %"; 1 de 8 → "12,50 %"; 4 de 7 → "57,14 %";
  1 de 6 → "16,67 %".
- Todas las historias "Hecho" → "100,00 %"; ninguna "Hecho" con historias → "0,00 %".
- La hora no importa: a las 00:00 y a las 23:59 del día de inicio el proyecto ya está "En curso"; a las 00:00 del día
  siguiente al fin, "Finalizado".
- Las fechas del proyecto se toman como días del calendario, sin zona horaria; "hoy" es el día del calendario en la
  zona local del servidor.
- Un proyecto con un Sprint activo antes de su fecha de inicio o después de su fecha de fin muestra igual el estado
  por fechas y el Sprint en curso: las dos cosas no se corrigen entre sí.
- Una historia que volvió al Backlog al cerrar un Sprint cuenta como "Pendiente" (su estado actual).
- El estado guardado del proyecto (un valor fijo) no se muestra ni se usa.
- Recargar la pantalla vuelve a calcular con la fecha y los datos del momento: el mismo proyecto puede pasar de "En
  curso" a "Finalizado" al cambiar el día.
- Los nombres de proyecto y los objetivos de Sprint se muestran escapados.
- Un proyecto con muchas historias (1000) muestra las cuatro cantidades sin listar las historias.

## Requisitos funcionales

- **FR-001**: El sistema MUST ofrecer la sección "Estado del proyecto" en `/estado`, que exige proyecto activo, y MUST
  responder solo a `GET` (y `HEAD`, que el enrutador de HU-20a atiende igual); cualquier otro método responde 405.
- **FR-002**: La pantalla MUST mostrar el nombre del proyecto activo y sus fechas de inicio y fin en dd/mm/aaaa.
- **FR-003**: La pantalla MUST mostrar el estado calculado con la regla R1, usando la fecha de hoy de R2.
- **FR-004**: La regla de estado MUST ser una función pura del dominio que recibe la fecha de inicio, la de fin y la de
  hoy, sin leer el reloj, para que otras historias (HU-09) la reutilicen.
- **FR-005**: La pantalla MUST mostrar "X de Y historias completadas (P %)" con la regla R3; sin historias, MUST
  mostrar "0 de 0 historias completadas (0,00 %)" sin error.
- **FR-006**: La regla de avance MUST ser una función pura del dominio que recibe las historias "Hecho" y el total y
  devuelve el porcentaje con punto y dos decimales ("50.00", "0.00"), para que HU-09 la reutilice.
- **FR-007**: La pantalla MUST mostrar las historias por estado según R4, en el orden Pendiente, En Sprint, En
  progreso y Hecho, incluidos los ceros.
- **FR-008**: Con Sprint activo, la pantalla MUST mostrar "Sprint N en curso", su objetivo y sus fechas, tal como los
  lee el Tablero (HU-14), sin modificar el Tablero.
- **FR-009**: Sin Sprint activo, la pantalla MUST mostrar "No hay un Sprint en curso" y el resto de la información
  igual.
- **FR-010**: Todas las historias y el Sprint mostrados MUST pertenecer al proyecto activo.
- **FR-011**: El nombre, las fechas y las cantidades MUST leerse en una sola lectura consistente; ante cualquier falla
  de lectura (proyecto, cantidades o Tablero), la pantalla MUST mostrar la página de error de HU-20a (500, o 503 si la
  base no responde) y MUST NOT mostrar datos parciales.
- **FR-012**: Si el proyecto activo ya no existe, la pantalla MUST responder la página 404 de HU-20a.
- **FR-013**: Sin proyecto activo MUST aplicarse la redirección de HU-20b, con regreso a "Estado del proyecto".
- **FR-014**: El sistema MUST NOT guardar resultados, usar el estado guardado del proyecto ni agregar migraciones.
- **FR-015**: Los textos de la pantalla MUST definirse en un único lugar; las respuestas de error MUST NOT incluir
  texto de consultas ni detalles internos.

## Entidades clave

- **Proyecto** (HU-01): nombre, fecha de inicio y fecha de fin.
- **Historia** (HU-02, HU-04, HU-14): proyecto y estado (Pendiente, En Sprint, En progreso o Hecho).
- **Tablero** (HU-14): el Sprint activo con su número, objetivo y fechas.
- **Estado del proyecto** (nuevo, no se guarda): "No iniciado", "En curso" o "Finalizado", calculado con las fechas.
- **Avance del proyecto** (nuevo, no se guarda): historias completadas, total y porcentaje.

## Criterios de éxito

- **SC-001**: En el 100% de las pruebas, el estado, las cantidades y el porcentaje coinciden con las cuentas a mano de
  "Datos de ejemplo", incluidos los dos bordes de fecha y los redondeos de los casos límite.
- **SC-002**: Ningún dato de otro proyecto aparece en la pantalla.
- **SC-003**: Ninguna falla de lectura muestra datos parciales.
- **SC-004**: El Scrum Master ve el estado y el avance en una sola acción (abrir la sección), y la pantalla responde en
  menos de 500 ms con la base local, 1000 historias y un Sprint activo con 50 historias.
- **SC-005**: Cada bloque tiene pruebas que fallaron antes del código (evidencia RED) y pasan después (evidencia
  GREEN), y la cobertura de `src/project` queda en `cobertura.txt` con al menos 80%.

## Supuestos y decisiones

- "Planificación de proyecto.pdf" no está en el repositorio; se usó el Issue (con sus datos de ejemplo) y las notas
  del equipo. Si al revisarlo aparece un conflicto, prevalece la planificación y se corrige esta especificación.
- `projects.status` vale 'Activo' en la base y "Active" en el dominio de HU-01; ninguna historia lo cambia. HU-12 no lo
  usa (notas del equipo).
- HU-01 ya exige que la fecha de fin sea posterior a la de inicio (FIX-HU-01), así que un proyecto siempre tiene al
  menos dos días "En curso".
- FIX-HU-04 garantiza un solo Sprint activo por proyecto; el Tablero muestra ese Sprint.
- Las cuatro cantidades suman el total del avance porque la base solo admite esos cuatro estados de historia.
- La fecha de hoy de las pruebas (28/10/2026) es la del Issue; el reloj de la pantalla se puede reemplazar en las
  pruebas.
- HU-08 también muestra historias por estado; si HU-12 entra primero, HU-08 puede reutilizar su lectura (notas del
  equipo). HU-09 reutiliza la regla de estado y la de avance (contrato en data-model.md).
- El Issue no dice qué tiene el Sprint 1 de «Alfa» ni cómo se reparten sus historias "Hecho" entre Sprints: las
  pruebas usan un Sprint 1 finalizado con 3 de las 4 "Hecho", y el Sprint 2 activo con la otra "Hecho", la "En Sprint"
  y la "En progreso". No cambia ninguna cuenta del escenario 1.

## Trazabilidad

| Escenario | Requisitos | Éxito | Prueba de pantalla / BDD | Prueba de dominio / integración |
|---|---|---|---|---|
| 1 Proyecto en curso | FR-002, FR-003, FR-005, FR-007, FR-008 | SC-001 | `TestHU12_ProyectoEnCurso`, `TestProjectStatusView`, TestEscenarioEstado_ProyectoEnCurso | `TestStatusOn`, `TestProgressPercentage`, `TestPostgresProjectStatusCatalog` |
| 2 No iniciado | FR-003, FR-004 | SC-001 | `TestHU12_EstadoPorFechas`, TestEscenarioEstado_ProyectoNoIniciado | `TestStatusOn` |
| 3 Finalizado | FR-003, FR-005 | SC-001 | `TestHU12_EstadoPorFechas`, TestEscenarioEstado_ProyectoFinalizado | `TestStatusOn`, `TestProgressPercentage` |
| 4 Empieza hoy | FR-003, FR-004 | SC-001 | `TestHU12_EstadoPorFechas` | `TestStatusOn` |
| 5 Termina hoy | FR-003, FR-004 | SC-001 | `TestHU12_EstadoPorFechas` | `TestStatusOn` |
| 6 Sin historias | FR-005, FR-006, FR-007 | SC-001 | `TestHU12_SinHistorias`, `TestProjectStatusView` | `TestProgressPercentage`, `TestPostgresProjectStatusCatalog` |
| 7 Sin Sprint en curso | FR-009 | — | `TestHU12_SinSprintEnCurso` | — (HU-14 ya probado) |
| 8 Otro proyecto | FR-010 | SC-002 | `TestHU12_OtroProyecto` | `TestPostgresProjectStatusCatalog` |
| 9 Sin proyecto | FR-013 | — | `TestHU12_SinProyecto`, TestEscenarioEstado_SinProyectoSeleccionado | — |
| 10 Error al consultar | FR-011, FR-012, FR-015 | SC-003 | `TestHU12_ErrorAlConsultar` | `TestPostgresProjectStatusCatalog` |
| (HU-20a) Solo lectura | FR-001 | — | `TestHU12_PostNoPermitido` | — |
| Rendimiento | — | SC-004 | `TestHU12_Rendimiento` | — |
| (Caso límite) Textos escapados | FR-002, FR-008 | — | `TestHU12_TextosEscapados` | — |
| Reglas reutilizables y sin persistencia | FR-004, FR-006, FR-014 | — | — | `TestStatusOn`, `TestProgressPercentage`, revisión en T023 (`git diff --stat main`) |
