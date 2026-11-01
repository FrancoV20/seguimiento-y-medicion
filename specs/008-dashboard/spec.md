# Especificación: HU-08 – Visualización de Dashboard del Proyecto

**Rama**: `HU-08`
**Creada**: 2026-10-28
**Estado**: Borrador
**Constitución**: v1.1.0
**Depende de**: HU-06, HU-07, HU-10, HU-14, HU-15, HU-20a, HU-20b
**Orden de implementación**: después de HU-10 (y por lo tanto de HU-06 y HU-15); HU-10 tiene que estar en `main`
antes de crear la rama (T001)
**Prioridad**: Media (Should Have) · **Valor**: 75 · **Estimación**: 8 SP

**Entrada**: "HU-08 / Visualización de Dashboard del Proyecto. Como Scrum Master, quiero visualizar un dashboard con
el estado general del proyecto (historias por estado, Sprint en curso, velocidad del equipo, desviación de esfuerzo y
defectos abiertos), con gráficos, para evaluar de un vistazo la salud del proyecto sin recorrer cada módulo por
separado."

## Historia de usuario

Como Scrum Master, quiero visualizar un dashboard con el estado general del proyecto (historias por estado, Sprint en
curso, velocidad del equipo, desviación de esfuerzo y defectos abiertos), con gráficos, para evaluar de un vistazo la
salud del proyecto sin recorrer cada módulo por separado.

> **Alcance**: HU-08 agrega la sección "Dashboard", de solo lectura, con cinco bloques del proyecto activo:
> historias por estado (gráfico y tabla), Sprint en curso, velocidad del equipo (gráfico por Sprint, acumulada y
> promedio), desviación de esfuerzo del último Sprint finalizado y defectos abiertos.
>
> **No recalcula lo que ya existe**: el Sprint en curso sale del Tablero (HU-14), la velocidad por Sprint del cálculo
> de calidad y avance (HU-10), la desviación del cálculo de métricas (HU-07) y los defectos de HU-06. Lo único nuevo
> del dominio es la velocidad acumulada y el promedio.
>
> **Por qué después de HU-10**: la velocidad por Sprint es la de HU-10, que a su vez lee el registro de cierre de
> HU-15 y los defectos de HU-06.

## Clarifications

### Session 2026-10-28

Decisiones del equipo registradas antes de `/speckit-clarify` (no se vuelven a preguntar):

- Q: ¿Cómo se ofrece la pantalla? → A: Sección "Dashboard" en `/dashboard`, que exige proyecto activo, de solo
  lectura (cualquier otro método → 405 de HU-20a), sin JavaScript, sin migración y sin guardar nada propio.
- Q: ¿Cómo son los gráficos? → A: Barras dibujadas en el servidor como imagen vectorial en línea, con descripción
  accesible ("role img" y etiqueta) y, al lado, una tabla con los mismos valores. La altura de cada barra es
  proporcional al valor máximo; con máximo 0, todas miden 0, sin dividir por cero. Sin dependencias nuevas.
- Q: ¿Qué historias cuentan en "historias por estado"? → A: Todas las del proyecto activo, agrupadas en Pendiente,
  En Sprint, En progreso y Hecho, en ese orden, aunque alguna dé 0.
- Q: ¿De dónde sale el Sprint en curso? → A: Del Tablero de HU-14, sin modificarlo: "Sprint N en curso", las
  cantidades En Sprint, En progreso y Hecho, y los Story Points planificados y completados que ya calcula el Tablero.
  Sin Sprint activo: "No hay un Sprint en curso".
- Q: ¿Cómo se calcula la velocidad? → A: Por cada Sprint finalizado, los Story Points completados de HU-10. El
  gráfico muestra como mucho los últimos 10 por número, del más antiguo al más reciente. Acumulada = suma de los
  completados de todos los Sprints finalizados; promedio = acumulada ÷ cantidad de Sprints finalizados, mitad hacia
  arriba a dos decimales (regla de HU-07), con coma ("10,50"). La suma y el promedio son una regla del dominio.
- Q: ¿Qué desviación se muestra? → A: La del último Sprint finalizado (el de mayor número), tal como la calcula
  HU-07, igual que HU-25: con signo, coma y dos decimales ("+20,00 %", "-8,50 %", "0,00 %"); "No disponible" tal
  cual. Se pide con la fuente de HU-25, que usa el cálculo de HU-07 sin guardar nada (confirmado en T001: el
  Dashboard no escribe en la tabla de métricas del Sprint).
- Q: ¿Qué defectos cuentan? → A: Los del proyecto activo en estado "Abierto". Texto: "1 defecto abierto", "N
  defectos abiertos" o "No hay defectos abiertos".
- Q: ¿Qué pasa sin datos? → A: Proyecto sin historias, sin Sprints y sin defectos: un único mensaje "Aún no hay datos
  suficientes para calcular métricas" en lugar de los gráficos. Sin Sprints finalizados: ese mensaje solo en los
  bloques de velocidad y desviación; los demás se muestran.
- Q: ¿Qué pasa si falla una lectura? → A: Cualquier falla de lectura o de cálculo (historias y defectos, Tablero,
  HU-10 o HU-07) muestra la página de error de HU-20a (500, o 503 si la base no responde), sin gráficos ni números
  parciales.
- Q: ¿Qué queda afuera? → A: Exportar (HU-09), detalle de Sprints (HU-16), filtros, hacer del Dashboard la página de
  inicio y defectos por severidad.

Pregunta de `/speckit-clarify`:

- Q: Cuando la desviación del último Sprint se calcula solo con algunas de sus historias terminadas, ¿se muestra el
  aviso de HU-07? → A: Sí. Debajo de la desviación se muestra el aviso de HU-07 tal cual ("Cálculo parcial: basado
  en X de Y historias"), solo cuando el cálculo es parcial, igual que en HU-25.

## Sección SDD

### Intención

Darle al Scrum Master, en una sola pantalla, una lectura rápida de la salud del proyecto: cuánto trabajo hay en cada
estado, cómo va el Sprint en curso, a qué ritmo entrega el equipo, qué tan precisas fueron las estimaciones del
último Sprint y cuántos defectos siguen abiertos.

### Alcance

- Sección "Dashboard" del proyecto activo, de solo lectura.
- Bloque 1, historias por estado: gráfico de barras y tabla.
- Bloque 2, Sprint en curso: número, cantidades por estado y Story Points planificados y completados.
- Bloque 3, velocidad del equipo: gráfico de barras por Sprint finalizado (últimos 10) y tabla, velocidad acumulada
  y promedio.
- Bloque 4, desviación de esfuerzo del último Sprint finalizado.
- Bloque 5, defectos abiertos.
- Estados vacíos: proyecto sin datos, sin Sprint en curso, sin Sprints finalizados, sin defectos abiertos.

### Fuera de alcance

- Exportar (HU-09) y detalle de Sprints anteriores (HU-16).
- Filtros por fecha o por Sprint.
- Que el Dashboard sea la página de inicio.
- Defectos por severidad u otros indicadores de defectos.
- Cambiar los cálculos de HU-07, HU-10 o HU-14, o sus pantallas.
- Guardar un histórico del Dashboard.

### Entradas

| Origen | Dato | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `GET /dashboard` | — | — | — | La pantalla no recibe parámetros |
| Contexto (HU-20b) | proyecto activo | identificador | sí | Nunca sale de la dirección ni de un formulario |

### Salidas esperadas

- Con datos: 200 con los cinco bloques. Cada gráfico tiene al lado una tabla con los mismos valores.
- Sin Sprint activo: el bloque 2 muestra "No hay un Sprint en curso".
- Sin Sprints finalizados: los bloques 3 y 4 se reemplazan por "Aún no hay datos suficientes para calcular métricas".
- Proyecto sin historias, sin Sprints y sin defectos: 200 con un único mensaje "Aún no hay datos suficientes para
  calcular métricas", sin gráficos.
- Falla al leer o calcular: la página de error de HU-20a (500 o 503), sin gráficos ni números parciales.
- Sin proyecto activo: redirección 303 de HU-20b a elegir proyecto, con regreso al Dashboard.
- Cualquier método distinto de `GET`: 405 de HU-20a.

### Reglas de negocio

- R1: historias por estado = todas las historias del proyecto activo contadas en Pendiente, En Sprint, En progreso y
  Hecho, en ese orden; un estado sin historias muestra 0.
- R2: el Sprint en curso es el Sprint activo del proyecto, con las cantidades y los Story Points que calcula el
  Tablero (HU-14): planificados = suma de todas sus historias estimadas; completados = suma de las "Hecho".
- R3: velocidad de un Sprint finalizado = Story Points completados que calcula HU-10 para ese Sprint (coinciden con
  la velocidad de HU-07).
- R4: el gráfico de velocidad muestra como mucho los últimos 10 Sprints finalizados por número, del más antiguo al
  más reciente.
- R5: velocidad acumulada = suma de las velocidades de todos los Sprints finalizados (no solo de los del gráfico);
  promedio = acumulada ÷ cantidad de Sprints finalizados, redondeado mitad hacia arriba a dos decimales (regla de
  HU-07), con coma.
- R6: desviación = la que calcula HU-07 para el último Sprint finalizado (mayor número), mostrada con signo solo si es
  positiva, coma, dos decimales y " %"; "No disponible" se muestra tal cual. Si HU-07 indica que el cálculo es
  parcial, debajo se muestra su aviso tal cual ("Cálculo parcial: basado en X de Y historias").
- R7: defectos abiertos = defectos del proyecto activo en estado "Abierto".
- R8: la altura de cada barra es proporcional al mayor valor del gráfico; si el mayor es 0, todas miden 0.
- R9: el Dashboard no guarda datos propios; cada apertura calcula con los datos del momento.

### Restricciones

- Sin migración ni tablas nuevas; sin dependencias nuevas; sin JavaScript.
- No se editan archivos de HU-07, HU-10, HU-14 ni HU-25. El único archivo compartido que se edita es el arranque del
  servidor (se suma un bloque de registro).
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- Los textos de la pantalla se definen en un único lugar.

### Condiciones de error

| Condición | Resultado | Respuesta |
|---|---|---|
| Proyecto sin historias, sin Sprints y sin defectos | Un único "Aún no hay datos suficientes para calcular métricas" | 200 |
| Sin Sprints finalizados | El mensaje anterior en los bloques de velocidad y desviación | 200 |
| Sin Sprint activo | "No hay un Sprint en curso" en el bloque 2 | 200 |
| Último Sprint sin horas estimadas | Desviación "No disponible" | 200 |
| Falla al leer historias, defectos, Tablero, calidad (HU-10) o métricas (HU-07) | Página de error de HU-20a, sin gráficos ni números parciales | 500 o 503 |
| Sin proyecto activo | Mensaje de HU-20b | 303 |
| Método distinto de `GET` | Página de HU-20a | 405 |

## Escenarios BDD

Los datos de ejemplo cierran entre sí: los escenarios 2, 3, 5, 7 y 9 describen el mismo proyecto «Alfa»
(ver "Datos de ejemplo"); los demás usan proyectos propios. Los escenarios se agrupan por bloque y conservan el
número del Issue.

### Bloque A – Estados vacíos

**Escenario 1: Proyecto sin datos suficientes**
- **Dado** un proyecto recién creado, sin historias ni Sprints
- **Cuando** ingreso al Dashboard
- **Entonces** veo "Aún no hay datos suficientes para calcular métricas" en lugar de un gráfico vacío o un error

**Escenario 4: Sin Sprint en curso**
- **Dado** un proyecto con 2 historias "Pendiente" y sin Sprints
- **Cuando** ingreso al Dashboard
- **Entonces** veo "No hay un Sprint en curso" y el resto del Dashboard se muestra igual

**Escenario 6: Sin Sprints finalizados**
- **Dado** un proyecto con 1 historia "Pendiente" y el Sprint 1 en curso con 1 historia "En Sprint", sin Sprints
  finalizados
- **Cuando** ingreso al Dashboard
- **Entonces** el bloque de velocidad y desviación muestra "Aún no hay datos suficientes para calcular métricas" y se
  ven las historias por estado y "Sprint 1 en curso"

### Bloque B – Historias y Sprint en curso

**Escenario 2: Historias por estado**
- **Dado** un proyecto con 2 historias "Pendiente", 1 "En Sprint", 2 "En progreso" y 3 "Hecho"
- **Cuando** ingreso al Dashboard
- **Entonces** veo un gráfico de historias por estado con esas cuatro cantidades y una tabla con los mismos valores

**Escenario 3: Sprint en curso**
- **Dado** que el Sprint 3 está en curso con 1 historia "En Sprint" de 1 Story Point, 2 "En progreso" de 3 y 5, y 1
  "Hecho" de 2
- **Cuando** ingreso al Dashboard
- **Entonces** veo "Sprint 3 en curso" con 1, 2 y 1 historias, 11 Story Points planificados y 2 completados

### Bloque C – Velocidad y desviación

**Escenario 5: Velocidad del equipo**
- **Dado** los Sprints 1 y 2 finalizados con 8 y 13 Story Points completados
- **Cuando** ingreso al Dashboard
- **Entonces** veo un gráfico de velocidad con una barra por Sprint, la velocidad acumulada en 21 Story Points y el
  promedio en 10,50 por Sprint

**Escenario 7: Desviación de esfuerzo del último Sprint**
- **Dado** el último Sprint finalizado con 10 horas estimadas y 12 horas reales en sus historias terminadas
- **Cuando** ingreso al Dashboard
- **Entonces** veo la desviación de ese Sprint como "+20,00 %"

**Escenario 8: Desviación no disponible**
- **Dado** el último Sprint finalizado con una historia terminada de 5 Story Points, sin horas estimadas
- **Cuando** ingreso al Dashboard
- **Entonces** veo la desviación como "No disponible" y la velocidad igual se muestra (5 Story Points)

### Bloque D – Defectos y aislamiento

**Escenario 9: Defectos abiertos**
- **Dado** un proyecto con 3 defectos "Abierto" y 2 "Resuelto"
- **Cuando** ingreso al Dashboard
- **Entonces** veo "3 defectos abiertos"

**Escenario 10: Datos de otro proyecto**
- **Dado** dos proyectos con historias, Sprints y defectos propios
- **Cuando** ingreso al Dashboard del proyecto activo
- **Entonces** solo veo datos de ese proyecto

### Bloque E – Pantalla

**Escenario 11: Gráficos sin JavaScript**
- **Dado** un Dashboard con datos
- **Cuando** lo abro con JavaScript desactivado
- **Entonces** veo los gráficos y, junto a cada uno, una tabla con los mismos valores

**Escenario 12: Error al leer los datos**
- **Dado** un fallo al leer los datos del proyecto
- **Cuando** ingreso al Dashboard
- **Entonces** veo la página de error de la aplicación y no veo gráficos ni números parciales

### Datos de ejemplo (proyecto «Alfa»)

| Historia | Story Points | Estado actual | Sprint | Detalle |
|---|---|---|---|---|
| A | 8 | Hecho | 1 (finalizado) | |
| P1 | 3 | Pendiente | 1 (finalizado) | No se terminó y volvió al Backlog al cerrar |
| B | 13 | Hecho | 2 (finalizado) | 10 horas estimadas; registros de esfuerzo de 8 y 4 horas (12 reales) |
| P2 | 2 | Pendiente | 2 (finalizado) | No se terminó y volvió al Backlog al cerrar |
| C | 1 | En Sprint | 3 (activo) | |
| D | 3 | En progreso | 3 (activo) | |
| E | 5 | En progreso | 3 (activo) | |
| F | 2 | Hecho | 3 (activo) | |

Cuentas:

- Por estado: Pendiente 2 (P1, P2), En Sprint 1 (C), En progreso 2 (D, E), Hecho 3 (A, B, F); total 8.
- Sprint 3 en curso: 1 + 2 + 1 = 4 historias; planificados 1 + 3 + 5 + 2 = 11; completados 2.
- Velocidad: Sprint 1 = 8 (A), Sprint 2 = 13 (B); acumulada 21; promedio 21 ÷ 2 = 10,50.
- Desviación del Sprint 2: (12 − 10) ÷ 10 × 100 = +20,00 %.
- Defectos: 3 "Abierto" y 2 "Resuelto" → "3 defectos abiertos".
- Gráfico de historias con 120 de alto: 2 → 80, 1 → 40, 2 → 80, 3 → 120. Gráfico de velocidad: 13 → 120;
  8 → 8 × 120 ÷ 13 = 73,85 → 74.

### Casos límite

- Un proyecto con un Sprint pero sin historias no está "sin datos": se ven los bloques, el gráfico de historias con
  las cuatro barras en 0 y su tabla con ceros.
- Sin defectos abiertos (ninguno o todos resueltos): "No hay defectos abiertos". Con uno: "1 defecto abierto".
- Más de 10 Sprints finalizados: con los Sprints 1 y 2 de 21 Story Points y los Sprints 3 a 12 de 5 Story Points, el
  gráfico muestra del Sprint 3 al 12 (10 barras), la acumulada es 42 + 50 = 92 y el promedio 92 ÷ 12 = 7,666… →
  "7,67". Se avisa "Se muestran los últimos 10 Sprints finalizados".
- El promedio se redondea mitad hacia arriba: 1 Story Point en 8 Sprints → 0,125 → "0,13"; 2 en 3 → "0,67".
- Un valor mayor que 0 nunca dibuja una barra de alto 0: mide al menos 1 (por ejemplo, 1 frente a 1000).
- Un Sprint finalizado sin historias "Hecho" tiene velocidad 0 y su barra mide 0; la tabla muestra 0.
- Un Sprint finalizado sin el registro de cierre de HU-15 (datos anteriores o cargados a mano) tiene velocidad 0 según
  HU-10, aunque HU-07 calcule otra; el Dashboard usa la de HU-10.
- Desviación negativa: 10 estimadas y 9,15 reales → "-8,50 %"; sin diferencia → "0,00 %", sin signo.
- Desviación parcial: en un último Sprint con dos historias terminadas, una con 10 horas estimadas y 12 reales y
  otra sin horas estimadas, se ve "+20,00 %" y debajo "Cálculo parcial: basado en 1 de 2 historias". En el Sprint 2
  de «Alfa» (una sola historia terminada, con horas) el aviso no aparece.
- En el escenario 8, HU-07 marca el cálculo como parcial porque la única historia terminada no tiene horas
  estimadas: se ve "No disponible" y debajo "Cálculo parcial: basado en 0 de 1 historias", que explica por qué.
- Si HU-07 rechaza el último Sprint (por ejemplo, una historia "Hecho" sin Story Points cargada a mano), se muestra la
  página de error, como cualquier falla de cálculo.
- Un defecto de otro proyecto nunca cuenta; tampoco sus historias ni sus Sprints.
- Recargar la pantalla vuelve a calcular con los datos actuales.
- Los números y las etiquetas de Sprint se muestran escapados; la pantalla no muestra textos ingresados por usuarios.

## Requisitos funcionales

- **FR-001**: El sistema MUST ofrecer la sección "Dashboard" en `/dashboard`, que exige proyecto activo, y MUST
  responder solo a `GET` (y `HEAD`, que el enrutador de HU-20a atiende igual); cualquier otro método responde 405.
- **FR-002**: El bloque de historias por estado MUST mostrar las cantidades de Pendiente, En Sprint, En progreso y
  Hecho del proyecto activo, en ese orden e incluyendo los ceros, en un gráfico y en una tabla.
- **FR-003**: Con Sprint activo, el bloque del Sprint en curso MUST mostrar "Sprint N en curso", las cantidades En
  Sprint, En progreso y Hecho, y los Story Points planificados y completados, tal como los calcula el Tablero
  (HU-14).
- **FR-004**: Sin Sprint activo, el bloque MUST mostrar "No hay un Sprint en curso" y el resto del Dashboard MUST
  mostrarse igual.
- **FR-005**: La velocidad de cada Sprint finalizado MUST ser la de Story Points completados de HU-10; el gráfico MUST
  mostrar como mucho los últimos 10 por número, del más antiguo al más reciente, con una tabla con los mismos
  valores.
- **FR-006**: El sistema MUST calcular la velocidad acumulada y el promedio con la regla R5 y mostrarlos como "N
  Story Points" y "X,XX Story Points por Sprint".
- **FR-007**: El sistema MUST mostrar la desviación del último Sprint finalizado calculada por HU-07, con el formato
  de R6; si el cálculo es parcial MUST mostrar debajo el aviso de HU-07 tal cual, y si no lo es MUST NOT mostrarlo.
- **FR-008**: El bloque de defectos MUST mostrar la cantidad de defectos "Abierto" del proyecto activo, con singular,
  plural o "No hay defectos abiertos".
- **FR-009**: Un proyecto sin historias, sin Sprints y sin defectos MUST mostrar un único mensaje "Aún no hay datos
  suficientes para calcular métricas" y MUST NOT mostrar gráficos.
- **FR-010**: Sin Sprints finalizados, los bloques de velocidad y desviación MUST reemplazarse por ese mensaje, y los
  demás bloques MUST mostrarse.
- **FR-011**: Todas las historias, Sprints y defectos mostrados MUST pertenecer al proyecto activo.
- **FR-012**: Cada gráfico MUST dibujarse en el servidor, sin JavaScript, con una descripción accesible y una tabla
  con los mismos valores al lado.
- **FR-013**: La altura de cada barra MUST ser proporcional al mayor valor del gráfico; con mayor 0 MUST medir 0 sin
  dividir por cero, y un valor mayor que 0 MUST medir al menos 1.
- **FR-014**: Ante cualquier falla de lectura o de cálculo, la pantalla MUST mostrar la página de error de HU-20a y
  MUST NOT mostrar gráficos ni números parciales.
- **FR-015**: El sistema MUST NOT agregar migraciones ni guardar datos propios; la única escritura es la que ya hace
  HU-07 al calcular la desviación, con los mismos valores.
- **FR-016**: Sin proyecto activo MUST aplicarse la redirección de HU-20b, con regreso al Dashboard.
- **FR-017**: El Dashboard MUST reutilizar los cálculos de HU-07, HU-10 y HU-14 sin cambiarlos; la velocidad
  acumulada y el promedio MUST ser una regla del dominio con prueba propia.
- **FR-018**: Los textos de la pantalla MUST definirse en un único lugar; las respuestas de error MUST NOT incluir
  texto de consultas ni detalles internos.

## Entidades clave

- **Historia** (HU-02): proyecto y estado (Pendiente, En Sprint, En progreso o Hecho).
- **Sprint** (HU-04): número, estado ("Activo" o "Finalizado") y proyecto.
- **Tablero** (HU-14): el Sprint activo con sus historias por estado y sus Story Points planificados y completados.
- **Calidad y avance del Sprint** (HU-10): Story Points completados de un Sprint finalizado.
- **Métricas del Sprint** (HU-07): desviación de esfuerzo de un Sprint finalizado.
- **Defecto** (HU-06): proyecto y estado ("Abierto" o "Resuelto").
- **Resumen de velocidad** (nuevo, no se guarda): velocidad acumulada y promedio de los Sprints finalizados.
- **Dashboard** (nuevo, no se guarda): los cinco bloques armados para la pantalla.

## Criterios de éxito

- **SC-001**: En el 100% de las pruebas, las cantidades, los Story Points, la acumulada, el promedio, la desviación y
  los defectos coinciden con el cálculo a mano sobre los datos de ejemplo, incluidos los redondeos de los casos
  límite.
- **SC-002**: En el 100% de las pruebas con Sprints cerrados, la velocidad de cada barra coincide con la de HU-10 y
  con la de HU-07 para el mismo Sprint.
- **SC-003**: Ningún dato de otro proyecto aparece en el Dashboard.
- **SC-004**: Ninguna falla de lectura o de cálculo muestra gráficos ni números parciales.
- **SC-005**: El Scrum Master ve los cinco bloques en una sola acción (abrir la sección), y la pantalla responde en
  menos de 500 ms con la base local, 20 Sprints finalizados, 200 historias y 100 defectos.
- **SC-006**: El 100% de los gráficos tiene una tabla con los mismos valores y la página no incluye JavaScript.
- **SC-007**: Cada bloque tiene pruebas que fallaron antes del código (evidencia RED) y pasan después (evidencia
  GREEN), y la cobertura de `src/metrics` queda en `cobertura.txt` con al menos 80%.

## Supuestos y decisiones

- Los nombres de HU-06 (`defects`), HU-10 (`QualityService`, `SprintQuality`, `QualityReader`), HU-15 (registro de
  cierre) y HU-25 (formato de la desviación) salen de sus documentos (`specs/006-defects`,
  `specs/010-quality-metrics`, `specs/015-sprint-close`, `specs/025-web-metrics`); se confirmaron en el código de
  `main` en T001 (research.md, "Verificación de T001").
- La planificación del proyecto (PDF) lista como dependencias HU-06, HU-07, HU-10 y HU-14; el Issue agrega HU-15,
  HU-20a y HU-20b. Se toman todas.
- El PDF nombra tres estados de historia (Pendiente, En progreso, Hecho); el esquema de `main` tiene cuatro desde
  HU-04 (agrega "En Sprint"). No es un conflicto: se muestran los cuatro para que la suma dé el total de historias.
- El escenario "Proyecto sin datos suficientes" del PDF ("sin Sprints finalizados") queda cubierto por los escenarios
  1 y 6 del Issue: en los dos casos se ve el mensaje en lugar de un gráfico vacío.
- "Último Sprint finalizado" es el de mayor número de Sprint.
- HU-07 calcula la desviación con las historias "Hecho" que quedan asignadas al Sprint, sus horas estimadas y la suma
  de sus registros de esfuerzo.
- El Tablero de HU-14 cuenta 0 Story Points para las historias sin estimar.
- Los escenarios 3, 4, 6 y 8 del Issue se escribieron con cantidades concretas (Story Points del Sprint en curso,
  historias de los proyectos sin Sprint o sin finalizados, Story Points del Sprint sin horas) para que sus datos
  cierren; el sentido de cada escenario no cambia.

## Trazabilidad

| Escenario | Requisitos | Éxito | Prueba de pantalla / BDD | Prueba de dominio / integración |
|---|---|---|---|---|
| 1 Sin datos suficientes | FR-009 | — | `TestHU08_SinDatos`, `TestEscenarioHU08_ProyectoSinDatosSuficientes` | `TestPostgresDashboardCatalog`, `TestBuildDashboardView` |
| 2 Historias por estado | FR-002, FR-012, FR-013 | SC-001, SC-006 | `TestHU08_HistoriasPorEstado`, `TestBuildDashboardView`, `TestEscenarioHU08_HistoriasPorEstado` | `TestPostgresDashboardCatalog`, `TestDashboardBarHeights` |
| 3 Sprint en curso | FR-003 | SC-001 | `TestHU08_SprintEnCurso` | — (HU-14 ya probado) |
| 4 Sin Sprint en curso | FR-004 | — | `TestHU08_SinSprintEnCurso` | — |
| 5 Velocidad del equipo | FR-005, FR-006, FR-017 | SC-001, SC-002 | `TestHU08_Velocidad`, `TestHU08_UltimosDiezSprints`, `TestEscenarioHU08_VelocidadDelEquipo` | `TestSummarizeVelocity`, `TestHU08_CoherenciaConHU07` |
| 6 Sin Sprints finalizados | FR-010 | — | `TestHU08_SinSprintsFinalizados` | `TestSummarizeVelocity` |
| 7 Desviación del último Sprint | FR-007 | SC-001 | `TestHU08_Desviacion`, `TestHU08_DesviacionParcial`, `TestBuildDashboardView`, `TestEscenarioHU08_DesviacionDelUltimoSprint` | `TestHU08_CoherenciaConHU07` (HU-07 ya probado) |
| 8 Desviación no disponible | FR-007 | SC-001 | `TestHU08_DesviacionNoDisponible`, `TestBuildDashboardView` | — |
| 9 Defectos abiertos | FR-008 | SC-001 | `TestHU08_DefectosAbiertos`, `TestDashboardTexts` | `TestPostgresDashboardCatalog` |
| 10 Datos de otro proyecto | FR-011 | SC-003 | `TestHU08_OtroProyecto` | `TestPostgresDashboardCatalog` |
| 11 Gráficos sin JavaScript | FR-012, FR-013 | SC-006 | `TestHU08_GraficosSinJavaScript` | `TestDashboardBarHeights`, `TestNewDashboardChart`, `TestNewDashboardChart_SinBarras` |
| 12 Error al leer | FR-014, FR-018 | SC-004 | `TestHU08_ErrorAlLeer` | `TestPostgresDashboardCatalog_ErrorDeLectura` |
| (HU-20b) Sin proyecto | FR-016 | — | `TestHU08_SinProyecto` | — |
| (HU-20a) Solo lectura | FR-001 | — | `TestHU08_PostNoPermitido` | — |
| Rendimiento | — | SC-005 | `TestHU08_Rendimiento` | — |
| (Caso límite) Sprint finalizado sin historias | FR-002, FR-013 | SC-001 | `TestHU08_SprintSinHistorias` | `TestDashboardBarHeights`, `TestSummarizeVelocity` |
| Sin persistencia propia | FR-015 | — | `TestHU08_NoEscribeSprintMetrics` | Revisión en T029 (`git diff --stat main`: solo archivos nuevos y `main.go`) |
