# Especificación: HU-09 – Exportación de Reportes en PDF

**Rama**: `HU-09`
**Creada**: 2026-10-28 · **Actualizada**: 2026-11-01 (implementación y trazabilidad con las pruebas reales)
**Estado**: Implementada (pendiente de revisión en el Pull Request)
**Constitución**: v1.1.0
**Depende de**: HU-06, HU-07, HU-08, HU-10, HU-12, HU-15, HU-17, HU-20a, HU-20b (verificación manual: HU-03/HU-24 y
HU-23)
**Orden de implementación**: después de HU-08, HU-10 y HU-12 (y por lo tanto de HU-06 y HU-15); las tres tienen que
estar en `main` antes de crear la rama (T001)
**Prioridad**: Media (Should Have; la DoD de la entrega final exige el reporte del proyecto en PDF) · **Valor**: 70 ·
**Estimación**: 8 SP

**Entrada**: "HU-09 / Exportación de Reportes en PDF. Como Scrum Master, quiero exportar en PDF el reporte de un
Sprint y el reporte del proyecto (historias planificadas y completadas, estimaciones, esfuerzo registrado, métricas y
defectos), para poder compartirlos con el Product Architect o archivarlos como evidencia del avance."

## Historia de usuario

Como Scrum Master, quiero exportar en PDF el reporte de un Sprint y el reporte del proyecto (historias planificadas y
completadas, estimaciones, esfuerzo registrado, métricas y defectos), para poder compartirlos con el Product
Architect o archivarlos como evidencia del avance.

> **Alcance**: HU-09 agrega la sección "Reportes", con la lista de Sprints del proyecto activo, un enlace "Exportar
> PDF" en cada uno y el botón "Exportar reporte del proyecto". Hay dos PDF:
>
> - el **reporte de un Sprint**, con siete partes: encabezado, historias, estimaciones, esfuerzo, métricas, defectos
>   y pie con "Página X de Y";
> - el **reporte del proyecto**, con seis partes: encabezado, avance, Sprints (con velocidad acumulada y promedio),
>   esfuerzo, defectos y pie.
>
> Un Sprint en curso, o un proyecto con un Sprint en curso, se puede exportar después de una advertencia, y el PDF
> sale marcado como parcial.
>
> **No recalcula lo que ya existe**: la velocidad, la desviación y el aviso parcial son los de HU-07; el avance, los
> Story Points y los defectos de cada Sprint, los de HU-10; la velocidad acumulada y el promedio, los de HU-08; el
> estado y el avance del proyecto, los de HU-12. Lo propio de HU-09 es leer los datos que se listan (historias,
> horas, esfuerzo por integrante y defectos) y armar los documentos.
>
> **Por qué después de HU-08, HU-10 y HU-12**: el reporte del proyecto usa la regla de velocidad acumulada de HU-08 y
> la de estado y avance de HU-12; los dos reportes usan HU-10, que a su vez lee el registro de cierre de HU-15 y los
> defectos de HU-06.

## Clarifications

### Session 2026-10-28

Decisiones del equipo registradas antes de `/speckit-clarify` (no se vuelven a preguntar):

- Q: ¿Qué se puede exportar? → A: El reporte de un Sprint y el reporte del proyecto activo (actualización del
  28/10: la DoD de la entrega final de la planificación dice "El sistema exporta el informe del proyecto en PDF").
  Quedan afuera los gráficos dentro del PDF, el envío por correo y guardar los PDF generados.
- Q: ¿Cómo es la pantalla? → A: Sección "Reportes" en `/reportes`, que exige proyecto activo. Lista los Sprints del
  proyecto activo (el activo primero, después los finalizados del número mayor al menor) con número, objetivo,
  estado y el enlace "Exportar PDF". Sin Sprints: "Aún no hay Sprints para exportar". El botón "Exportar reporte del
  proyecto" está siempre, aunque no haya Sprints.
- Q: ¿Cómo se descarga el reporte de un Sprint? → A: Con `GET /reportes/sprints/{id}/pdf`. Un identificador
  inválido, inexistente o de otro proyecto da la página 404 de HU-20a, sin leer nada más. Un Sprint finalizado da el
  PDF como descarga, con el nombre `reporte-sprint-N.pdf` y sin guardarse en la caché. Cualquier otro método da 405.
- Q: ¿Qué pasa con un Sprint en curso? → A: Sin la confirmación (`?confirmar=si`) se muestra una página de
  advertencia, "El Sprint N sigue en curso: el reporte puede estar incompleto", con "Generar igual" (la misma
  dirección con `?confirmar=si`) y "Cancelar" (vuelve a "Reportes"). Con la confirmación se descarga el PDF marcado
  "Reporte parcial: Sprint en curso". Cualquier otro valor de `confirmar` se trata como ausente. Sin JavaScript.
- Q: ¿Qué contiene el PDF de un Sprint y en qué orden? → A:
  1. Encabezado: proyecto, "Sprint N", objetivo, fechas (dd/mm/aaaa), estado y fecha y hora de generación.
  2. Historias: título, estado (en un Sprint finalizado, el del registro de cierre; las no "Hecho" llevan "volvió al
     Backlog"), Story Points y horas estimadas ("Sin estimar" si faltan) y horas reales; debajo, "X de Y historias
     completadas (P %)".
  3. Estimaciones: Story Points planificados y completados, y horas estimadas totales (suma de las informadas, con
     "N historia(s) sin horas estimadas" si corresponde).
  4. Esfuerzo: horas registradas en total y por integrante, ordenado por nombre.
  5. Métricas: velocidad, desviación ("+6,25 %" o "No disponible") y el aviso parcial de HU-07 tal cual. En un Sprint
     en curso: "Se calculan al cerrar el Sprint".
  6. Defectos: los detectados en el Sprint (descripción, severidad, estado, historia y Sprint de resolución si
     tiene) y "N defectos detectados y M resueltos en este Sprint" con los conteos de HU-10. Sin ninguno: "No se
     registraron defectos en este Sprint". En un Sprint en curso, la lista y el conteo de detectados hasta el
     momento.
  7. Pie: "Página X de Y" en todas las páginas.
- Q: ¿De dónde sale cada dato del reporte de un Sprint? → A: No se recalcula nada:
  - historias de un Sprint finalizado: registro de cierre de HU-15, con el título y las horas estimadas actuales de
    la historia; de un Sprint en curso, las asignaciones actuales con su estado y sus Story Points;
  - horas reales: suma de todos los registros de esfuerzo de cada historia, el mismo criterio que HU-07;
  - velocidad, desviación y aviso parcial: HU-07 (que vuelve a escribir los mismos valores en la tabla de métricas
    del Sprint, como en HU-25 y HU-08);
  - porcentaje de avance, Story Points planificados y completados, y defectos detectados y resueltos: HU-10;
  - lo demás (encabezado, historias, horas, esfuerzo por integrante y lista de defectos) se lee junto, en una sola
    lectura consistente.
- Q: ¿Cómo se descarga el reporte del proyecto? → A: Con `GET /reportes/proyecto/pdf`; el proyecto es siempre el del
  contexto, nunca sale de la dirección. Nombre `reporte-proyecto-N.pdf` (N = identificador del proyecto), con los
  mismos encabezados que el de Sprint. Cualquier otro método da 405.
- Q: ¿Qué pasa si el proyecto tiene un Sprint en curso? → A: Sin `?confirmar=si`, la advertencia "El proyecto tiene un
  Sprint en curso (Sprint N): el reporte puede estar incompleto", con "Generar igual" y "Cancelar", igual que la del
  Sprint. Con la confirmación, el PDF lleva "Reporte parcial: hay un Sprint en curso". Sin Sprint activo se genera
  directo, sin advertencia.
- Q: ¿Qué contiene el PDF del proyecto y en qué orden? → A:
  1. Encabezado: nombre, fechas, estado por fechas (HU-12), integrantes (orden alfabético) y fecha y hora de
     generación.
  2. Avance: historias por estado (Pendiente, En Sprint, En progreso y Hecho, con ceros) y "X de Y historias
     completadas (P %)" con la regla de HU-12.
  3. Sprints: una fila por Sprint en orden de número, con historias completadas y planificadas, Story Points
     completados y planificados, velocidad, desviación (con "(cálculo parcial)" si HU-07 lo marca) y defectos
     detectados y resueltos; el activo, con "En curso: se calcula al cerrar el Sprint". Debajo, la velocidad
     acumulada y el promedio de los finalizados (HU-08). Sin Sprints: "Aún no hay Sprints".
  4. Esfuerzo: horas estimadas totales del proyecto (con "N historia(s) sin horas estimadas"), horas registradas
     totales y por integrante.
  5. Defectos: "N defectos registrados: A abiertos y R resueltos" (singular y plural correctos) y la lista de los
     abiertos (descripción, severidad, historia y Sprint de detección). Sin defectos: "No se registraron defectos en
     este proyecto".
  6. Pie: "Página X de Y".
- Q: ¿De dónde sale cada dato del reporte del proyecto? → A: No se recalcula nada: estado y avance con la regla de
  HU-12; por cada Sprint finalizado, HU-10 y HU-07 (que reescribe `sprint_metrics`, aceptado); acumulada y promedio
  con la regla de HU-08. Lo propio (encabezado, integrantes, historias por estado, horas estimadas y registradas,
  esfuerzo por integrante y defectos) se lee junto, en una sola lectura consistente, con el mismo puerto que el
  reporte del Sprint.
- Q: ¿Cómo se arma el PDF? → A: Con un generador propio, sin bibliotecas externas: PDF 1.4, A4, fuentes estándar
  Helvetica y Helvetica-Bold, acentos y "ñ" correctos, textos largos cortados en varias líneas, saltos de página
  automáticos con el encabezado de la tabla repetido. El documento se arma completo antes de enviarlo; ante cualquier
  falla se muestra la página de error y no se descarga nada.

Pregunta de `/speckit-clarify`:

- Q: En el PDF de un Sprint en curso, ¿qué se muestra en "X de Y historias completadas (P %)" y en los Story Points
  planificados y completados, que en un Sprint finalizado vienen de HU-10? → A: También "Se calculan al cerrar el
  Sprint", igual que la velocidad y la desviación. No se recalcula nada; la tabla de historias, las horas estimadas
  totales, el esfuerzo y los defectos detectados hasta el momento sí se muestran con los datos del momento.

## Sección SDD

### Intención

Que el Scrum Master tenga, en archivos que puede compartir o archivar, una foto completa de un Sprint (qué se
planificó y qué se terminó, con cuánto esfuerzo, qué métricas dio y qué defectos aparecieron) y del proyecto entero
(avance, Sprints, velocidad, esfuerzo y defectos), como pide la DoD de la entrega final.

### Alcance

- Sección "Reportes" con la lista de Sprints del proyecto activo, el enlace "Exportar PDF" y el botón "Exportar
  reporte del proyecto".
- Descarga del PDF de un Sprint finalizado; advertencia y PDF parcial de un Sprint en curso.
- Descarga del PDF del proyecto; advertencia y PDF parcial si hay un Sprint en curso.
- Contenido de los dos PDF, en el orden de "Clarifications".
- Estados vacíos: proyecto sin Sprints ni historias, Sprint sin defectos, Sprint sin historias, historias sin
  estimar, proyecto sin defectos.
- Generador de PDF propio, reutilizable.

### Fuera de alcance

- Gráficos dentro del PDF (los gráficos son de HU-08).
- Enviar los PDF por correo y guardar los PDF generados.
- Reportes de varios proyectos o de un rango de fechas.
- Cambiar los cálculos de HU-07, HU-08, HU-10 o HU-12, o sus pantallas.
- Elegir qué partes incluir, el idioma, el tamaño de página o la orientación.

### Entradas

| Origen | Dato | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `GET /reportes` | — | — | — | La pantalla no recibe parámetros |
| `GET /reportes/sprints/{id}/pdf` | `id` (en la dirección) | identificador numérico | sí | Sprint del proyecto activo; inválido, inexistente o ajeno = 404 |
| `GET /reportes/sprints/{id}/pdf` y `GET /reportes/proyecto/pdf` | `confirmar` | texto | no | Solo "si" confirma; cualquier otro valor = ausente |
| Contexto (HU-20b) | proyecto activo | identificador | sí | Nunca sale de la dirección ni de un formulario |

### Salidas esperadas

- `GET /reportes`: 200 con la lista de Sprints (el activo primero, después los finalizados del mayor al menor), o
  con "Aún no hay Sprints para exportar"; en los dos casos, el botón "Exportar reporte del proyecto".
- Sprint finalizado: 200 con el PDF como descarga (`reporte-sprint-N.pdf`), sin caché.
- Sprint en curso sin confirmar: 200 con la página de advertencia ("Generar igual" y "Cancelar"); no se genera nada.
- Sprint en curso confirmado: 200 con el PDF parcial como descarga.
- Sprint inválido, inexistente o de otro proyecto: 404 de HU-20a, sin archivo.
- Proyecto sin Sprint en curso: 200 con el PDF del proyecto como descarga (`reporte-proyecto-N.pdf`), sin caché.
- Proyecto con un Sprint en curso sin confirmar: 200 con la página de advertencia; con confirmación, el PDF parcial.
- Falla al leer, al calcular o al armar cualquiera de los dos PDF: la página de error de HU-20a (500 o 503), sin
  archivo.
- Sin proyecto activo: redirección 303 de HU-20b a elegir proyecto, con regreso a la dirección pedida.
- Cualquier método distinto de `GET`: 405 de HU-20a.

### Reglas de negocio

Reporte de un Sprint:

- R1: solo se exportan Sprints del proyecto activo.
- R2: historias de un Sprint finalizado = las del registro de cierre (HU-15), con el estado y los Story Points que
  tenían al cerrar; las que no terminaron en "Hecho" se muestran con su estado final y "volvió al Backlog". De un
  Sprint en curso = las asignadas en este momento, con su estado y sus Story Points actuales. Las horas estimadas
  son las actuales de cada historia. Se listan en el orden en que se crearon.
- R3: horas reales de una historia = suma de todos sus registros de esfuerzo (mismo criterio que HU-07).
- R4: horas estimadas totales = suma de las horas estimadas informadas; si hay historias sin horas estimadas, se
  avisa "1 historia sin horas estimadas" o "N historias sin horas estimadas".
- R5: horas registradas = suma de las horas reales de las historias del Sprint; por integrante = las mismas horas
  agrupadas por quien las registró, ordenadas por nombre. La suma por integrante es igual al total.
- R6: "X de Y historias completadas (P %)", los Story Points planificados y completados y los conteos de defectos
  son los de HU-10; el porcentaje se muestra con coma y dos decimales, o "No disponible".
- R7: velocidad, desviación y aviso parcial son los de HU-07. La desviación se muestra con signo "+" solo si es
  positiva, coma, dos decimales y " %"; "No disponible" tal cual. El aviso parcial se muestra tal cual, solo si HU-07
  indica que el cálculo es parcial.
- R8: los defectos listados son los del proyecto cuyo Sprint de detección es el Sprint. El resumen dice "N defectos
  detectados y M resueltos en este Sprint" (singular con 1: "1 defecto detectado", "1 resuelto"); los resueltos
  pueden incluir defectos detectados en Sprints anteriores. Sin detectados ni resueltos: "No se registraron
  defectos en este Sprint".
- R9: en un Sprint en curso no se consulta a HU-07 ni a HU-10 (solo calculan Sprints finalizados). En lugar del
  avance ("X de Y historias completadas (P %)"), de los Story Points planificados y completados y de la parte de
  métricas se muestra "Se calculan al cerrar el Sprint". El resumen de defectos cuenta los detectados hasta el
  momento: "1 defecto detectado hasta el momento", "N defectos detectados hasta el momento", o "No se registraron
  defectos en este Sprint" si no hay. La tabla de historias, las horas estimadas totales y el esfuerzo se muestran
  con los datos del momento.

Reporte del proyecto:

- R12: el reporte es siempre del proyecto activo. Si tiene un Sprint en curso, el PDF lleva "Reporte parcial: hay un
  Sprint en curso" (después de la advertencia); si no, se genera directo.
- R13: el estado del proyecto ("No iniciado", "En curso" o "Finalizado") es el que calcula HU-12 con sus fechas y la
  fecha de generación. Los integrantes se listan por nombre, en orden alfabético.
- R14: historias por estado = todas las historias del proyecto, en su estado actual, contadas en Pendiente, En
  Sprint, En progreso y Hecho, en ese orden y con ceros. "X de Y historias completadas (P %)" usa la regla de HU-12:
  X = "Hecho", Y = total; con 0 historias, "0 de 0 historias completadas (0,00 %)".
- R15: una fila por Sprint, por número ascendente. De un Sprint finalizado: historias completadas y planificadas,
  Story Points completados y planificados y defectos detectados y resueltos de HU-10; velocidad y desviación de
  HU-07, con " (cálculo parcial)" si HU-07 lo marca. De un Sprint en curso: "En curso: se calcula al cerrar el
  Sprint". Sin Sprints: "Aún no hay Sprints".
- R16: velocidad acumulada y promedio = la regla de HU-08 aplicada a las velocidades de los Sprints finalizados
  ("Velocidad acumulada: N Story Points" y "Promedio: X,XX Story Points por Sprint"; sin finalizados, "Promedio: No
  disponible").
- R17: horas estimadas totales = suma de las informadas de todas las historias del proyecto, con el aviso de R4;
  horas registradas = suma de todos los registros de esfuerzo de las historias del proyecto; por integrante, las
  mismas horas agrupadas por quien las registró, ordenadas por nombre.
- R18: defectos = todos los del proyecto: "N defectos registrados: A abiertos y R resueltos" (singular con 1: "1
  defecto registrado", "1 abierto", "1 resuelto"); debajo, la lista de los abiertos con descripción, severidad,
  historia y Sprint de detección. Sin defectos: "No se registraron defectos en este proyecto". Sin abiertos: "No hay
  defectos abiertos".

Comunes:

- R10: cada PDF se arma completo antes de enviarlo; si algo falla no se envía ningún byte del archivo.
- R11: cada exportación lee los datos del momento; nada se guarda, salvo la reescritura de los mismos valores que ya
  hace HU-07 en la tabla de métricas de cada Sprint finalizado.

### Restricciones

- Sin migración ni tablas nuevas; sin dependencias nuevas; sin JavaScript.
- No se editan archivos de HU-07, HU-08, HU-10 ni HU-12. El único archivo compartido que se edita es el arranque del
  servidor (se suma un bloque de registro).
- Ningún mensaje ni los PDF exponen consultas, rutas ni valores de configuración.
- Los textos de la pantalla y de los PDF se definen en un único lugar cada uno.
- Los PDF se abren en los lectores habituales (navegador y lector de escritorio) sin advertencias de archivo dañado.

### Condiciones de error

| Condición | Resultado | Respuesta |
|---|---|---|
| Proyecto sin Sprints | "Aún no hay Sprints para exportar" y el botón del proyecto | 200 |
| `id` inválido (vacío, no numérico, cero, negativo, con ceros a la izquierda o mayor que el máximo) | Página 404 de HU-20a, sin leer nada más | 404 |
| Sprint inexistente o de otro proyecto | Página 404 de HU-20a, sin archivo | 404 |
| Sprint en curso, o proyecto con un Sprint en curso, sin `confirmar=si` | Página de advertencia, sin archivo | 200 |
| Falla al leer los datos, en HU-07, HU-08, HU-10, HU-12 o al armar el PDF | Página de error de HU-20a, sin bytes del PDF | 500 o 503 |
| Sin proyecto activo | Mensaje de HU-20b | 303 |
| Método distinto de `GET` | Página de HU-20a | 405 |

## Escenarios BDD

Los datos de ejemplo cierran entre sí. Del reporte de un Sprint: los escenarios 1 a 6 usan el Sprint 2 de «Alfa»;
el 7, el Sprint 1; el 8 a 10, el Sprint 3. Del reporte del proyecto: el 14 a 17 usan «Alfa» entero; el 11 y el 18,
«Beta». Se agrupan por bloque y conservan el número del Issue.

### Bloque A – Sprint finalizado

**Escenario 1: Exportación exitosa de un Sprint finalizado**
- **Dado** el Sprint 2 finalizado de «Alfa»
- **Cuando** en "Reportes" presiono "Exportar PDF" en el Sprint 2
- **Entonces** se descarga el archivo "reporte-sprint-2.pdf" con el proyecto, el Sprint, su objetivo, sus fechas y la
  fecha de generación

**Escenario 2: Historias planificadas y completadas**
- **Dado** el Sprint 2 finalizado de «Alfa»
- **Cuando** exporto su reporte
- **Entonces** el PDF lista las 4 historias con su estado al cerrar, indica que «Reportes» volvió al Backlog y
  muestra "3 de 4 historias completadas (75,00 %)"

**Escenario 3: Estimaciones**
- **Dado** el Sprint 2 finalizado de «Alfa»
- **Cuando** exporto su reporte
- **Entonces** el PDF muestra los Story Points y las horas estimadas de cada historia, "Sin estimar" en las horas de
  «Perfil», 18 Story Points planificados, 10 completados y 28 horas estimadas en total

**Escenario 4: Esfuerzo registrado**
- **Dado** el Sprint 2 finalizado de «Alfa»
- **Cuando** exporto su reporte
- **Entonces** el PDF muestra las horas reales de cada historia, 24 horas registradas en total y el esfuerzo por
  integrante: Ana 17 h y Beto 7 h

**Escenario 5: Métricas del Sprint**
- **Dado** el Sprint 2 finalizado de «Alfa»
- **Cuando** exporto su reporte
- **Entonces** el PDF muestra la velocidad en 10 Story Points, la desviación de esfuerzo en "+6,25 %" y el aviso
  "Cálculo parcial: basado en 2 de 3 historias"

**Escenario 6: Defectos del Sprint**
- **Dado** el Sprint 2 finalizado de «Alfa»
- **Cuando** exporto su reporte
- **Entonces** el PDF lista los 2 defectos detectados en el Sprint con su descripción, severidad, estado e historia,
  y muestra "2 defectos detectados y 1 resuelto en este Sprint"

**Escenario 7: Sprint sin defectos**
- **Dado** el Sprint 1 finalizado de «Alfa», sin defectos detectados ni resueltos
- **Cuando** exporto su reporte
- **Entonces** el PDF muestra "No se registraron defectos en este Sprint"

### Bloque B – Sprint en curso

**Escenario 8: Advertencia al exportar un Sprint en curso**
- **Dado** que el Sprint 3 de «Alfa» está en curso
- **Cuando** presiono "Exportar PDF" en el Sprint 3
- **Entonces** veo "El Sprint 3 sigue en curso: el reporte puede estar incompleto" con las opciones "Generar igual" y
  "Cancelar", y todavía no se genera el PDF

**Escenario 9: Generar igual el reporte de un Sprint en curso**
- **Dado** la advertencia del Sprint 3 en curso
- **Cuando** presiono "Generar igual"
- **Entonces** se descarga el PDF marcado como "Reporte parcial: Sprint en curso", con las historias en su estado
  actual y el avance, los Story Points planificados y completados y las métricas como "Se calculan al cerrar el
  Sprint"

**Escenario 10: Cancelar**
- **Dado** la advertencia del Sprint 3 en curso
- **Cuando** presiono "Cancelar"
- **Entonces** vuelvo a "Reportes" y no se genera nada

### Bloque C – Pantalla y errores

**Escenario 11: Sin Sprints**
- **Dado** el proyecto «Beta», sin Sprints
- **Cuando** abro "Reportes"
- **Entonces** veo "Aún no hay Sprints para exportar" y el botón "Exportar reporte del proyecto"

**Escenario 12: Sprint inexistente o de otro proyecto**
- **Dado** un Sprint que no existe o que pertenece a otro proyecto
- **Cuando** pido exportar su reporte
- **Entonces** veo la página 404 de la aplicación y no se genera ningún archivo

**Escenario 13: Error al generar**
- **Dado** un fallo al leer los datos
- **Cuando** exporto un reporte de Sprint o del proyecto
- **Entonces** veo la página de error de la aplicación y no se descarga un archivo incompleto

### Bloque D – Reporte del proyecto

**Escenario 14: Advertencia al exportar un proyecto con un Sprint en curso**
- **Dado** el proyecto «Alfa», con el Sprint 3 en curso
- **Cuando** en "Reportes" presiono "Exportar reporte del proyecto"
- **Entonces** veo "El proyecto tiene un Sprint en curso (Sprint 3): el reporte puede estar incompleto" con las
  opciones "Generar igual" y "Cancelar", y todavía no se genera el PDF

**Escenario 15: Encabezado del reporte del proyecto**
- **Dado** la advertencia del proyecto «Alfa»
- **Cuando** presiono "Generar igual"
- **Entonces** se descarga "reporte-proyecto-N.pdf" (N es el identificador del proyecto), marcado como "Reporte
  parcial: hay un Sprint en curso", con el nombre del proyecto, sus fechas (18/08/2026 al 30/11/2026), su estado "En
  curso", sus integrantes (Ana y Beto) y la fecha de generación

**Escenario 16: Avance y Sprints del proyecto**
- **Dado** el reporte del proyecto «Alfa»
- **Cuando** lo abro
- **Entonces** veo las historias por estado (Pendiente 1, En Sprint 1, En progreso 1, Hecho 4), "4 de 7 historias
  completadas (57,14 %)" y una fila por Sprint:
  - Sprint 1: 1 de 1 historias, 3 de 3 Story Points, velocidad 3, desviación "0,00 %", 0 defectos detectados y 0
    resueltos;
  - Sprint 2: 3 de 4 historias, 10 de 18 Story Points, velocidad 10, desviación "+6,25 %" (cálculo parcial), 2
    defectos detectados y 1 resuelto;
  - Sprint 3: "En curso: se calcula al cerrar el Sprint";

  y la velocidad acumulada en 13 Story Points con un promedio de 6,50 por Sprint

**Escenario 17: Esfuerzo y defectos del proyecto**
- **Dado** el reporte del proyecto «Alfa»
- **Cuando** lo abro
- **Entonces** veo 38 horas estimadas ("2 historias sin horas estimadas"), 30 horas registradas (Ana 23 h y Beto 7 h),
  "3 defectos registrados: 2 abiertos y 1 resuelto" y la lista de los 2 defectos abiertos

**Escenario 18: Proyecto sin Sprints ni historias**
- **Dado** el proyecto «Beta», sin Sprints ni historias
- **Cuando** presiono "Exportar reporte del proyecto"
- **Entonces** se descarga el PDF sin advertencia, con "Aún no hay Sprints", "0 de 0 historias completadas (0,00 %)",
  "0 horas registradas en total" y "No se registraron defectos en este proyecto"

### Datos de ejemplo

Proyecto «Alfa», del 18/08/2026 al 30/11/2026, integrantes Ana y Beto. Fecha y hora de generación de las pruebas:
28/10/2026 10:30 (con esa fecha, HU-12 da "En curso").

| Sprint | Estado | Objetivo | Fechas |
|---|---|---|---|
| 1 | Finalizado | Dar de alta usuarios | 18/08/2026 al 31/08/2026 |
| 2 | Finalizado | Gestionar perfiles y reportes | 01/09/2026 al 14/09/2026 |
| 3 | Activo (en curso) | Exportar e informar | 15/09/2026 al 28/09/2026 |

Historias (en el orden en que se crearon):

| Historia | Sprint | Story Points | Horas estimadas | Estado | Registros de esfuerzo | Horas reales |
|---|---|---|---|---|---|---|
| Alta | 1 | 3 | 4 | Hecho al cerrar (hoy "Hecho") | Ana 4 | 4 |
| Login | 2 | 5 | 10 | Hecho al cerrar (hoy "Hecho") | Ana 8, Beto 3 | 11 |
| Tablero | 2 | 3 | 6 | Hecho al cerrar (hoy "Hecho") | Ana 6 | 6 |
| Perfil | 2 | 2 | Sin estimar | Hecho al cerrar (hoy "Hecho") | Ana 3, Beto 1 | 4 |
| Reportes | 2 | 8 | 12 | "En progreso" al cerrar; volvió al Backlog (hoy "Pendiente") | Beto 3 | 3 |
| Exportar | 3 | 5 | 6 | En progreso (actual) | Ana 2 | 2 |
| Ayuda | 3 | Sin estimar | Sin estimar | En Sprint (actual) | — | 0 |

Defectos:

| Defecto | Severidad | Historia | Detectado en | Estado | Resuelto en |
|---|---|---|---|---|---|
| "El login no valida la contraseña" | Alta | Login | Sprint 2 | Resuelto | Sprint 2 |
| "La foto de perfil no se guarda" | Media | Perfil | Sprint 2 | Abierto | — |
| "El PDF sale en blanco" | Crítica | Exportar | Sprint 3 | Abierto | — |

Proyecto «Beta»: sin historias ni Sprints (en las pruebas: del 01/10/2026 al 31/12/2026, integrante Ana).

Cuentas del Sprint 2:

- Historias: 4 en el registro de cierre; "Hecho": Login, Tablero y Perfil → "3 de 4 historias completadas
  (75,00 %)" (3 ÷ 4 × 100 = 75).
- Story Points: planificados 5 + 3 + 2 + 8 = 18; completados 5 + 3 + 2 = 10.
- Horas estimadas: 10 + 6 + 12 = 28; «Perfil» sin estimar → "1 historia sin horas estimadas".
- Horas reales: 11 + 6 + 4 + 3 = 24. Por integrante: Ana 8 + 6 + 3 = 17 y Beto 3 + 1 + 3 = 7; 17 + 7 = 24. Las 4
  horas de Ana en «Alta» (Sprint 1) y las 2 en «Exportar» (Sprint 3) no cuentan.
- HU-07 (historias "Hecho" asignadas): velocidad 5 + 3 + 2 = 10, igual a los completados de HU-10. Usa las que tienen
  horas estimadas y reales: Login (10 y 11) y Tablero (6 y 6) → 16 estimadas y 17 reales; (17 − 16) ÷ 16 × 100 = 6,25
  → "+6,25 %". «Perfil» no tiene horas estimadas → "Cálculo parcial: basado en 2 de 3 historias".
- Defectos (HU-10): detectados en el Sprint 2, los dos primeros → 2; resueltos en el Sprint 2, el primero → 1. Texto:
  "2 defectos detectados y 1 resuelto en este Sprint".

Cuentas del Sprint 1 (escenario 7): 1 de 1 (100,00 %), 3 y 3 Story Points, 4 horas estimadas y 4 registradas (Ana
4 h), velocidad 3, desviación (4 − 4) ÷ 4 = "0,00 %" sin aviso parcial (la única historia "Hecho" tiene horas
estimadas y reales: 1 de 1), "No se registraron defectos en este Sprint".

Cuentas del Sprint 3 (escenarios 8 y 9, en curso): 2 historias; «Exportar» "En progreso" con 5, 6 y 2; «Ayuda» "En
Sprint" con "Sin estimar", "Sin estimar" y 0. Avance, Story Points planificados y completados y métricas: "Se
calculan al cerrar el Sprint". Horas estimadas 6 ("1 historia sin horas estimadas"); 2 horas registradas (Ana 2 h);
"1 defecto detectado hasta el momento".

Cuentas del proyecto «Alfa» (escenarios 14 a 17):

- Historias por estado actual: Pendiente 1 (Reportes), En Sprint 1 (Ayuda), En progreso 1 (Exportar), Hecho 4 (Alta,
  Login, Tablero, Perfil); total 7. 4 ÷ 7 × 100 = 57,142… → "4 de 7 historias completadas (57,14 %)".
- Sprints: el 1 y el 2 con los números de arriba; el 3, "En curso: se calcula al cerrar el Sprint". Velocidad
  acumulada 3 + 10 = 13; promedio 13 ÷ 2 = 6,50.
- Horas estimadas: 4 + 10 + 6 + 12 + 6 = 38; «Perfil» y «Ayuda» sin estimar → "2 historias sin horas estimadas".
- Horas registradas: 4 + 11 + 6 + 4 + 3 + 2 = 30. Ana 4 + 8 + 6 + 3 + 2 = 23 y Beto 3 + 1 + 3 = 7; 23 + 7 = 30.
- Defectos: 3 registrados, 2 abiertos ("La foto de perfil no se guarda", detectado en el Sprint 2, y "El PDF sale en
  blanco", detectado en el Sprint 3) y 1 resuelto → "3 defectos registrados: 2 abiertos y 1 resuelto".
- Estado: 28/10/2026 está entre el 18/08/2026 y el 30/11/2026 → "En curso" (HU-12).

Cuentas del proyecto «Beta» (escenarios 11 y 18): historias por estado 0, 0, 0 y 0; "0 de 0 historias completadas
(0,00 %)"; "Aún no hay Sprints"; "0 horas estimadas en total"; "0 horas registradas en total" sin integrantes
listados; "No se registraron defectos en este proyecto". Sin Sprint en curso: se descarga sin advertencia.

### Casos límite

Reporte de un Sprint:

- Un `id` vacío, no numérico, negativo, cero, con ceros a la izquierda o mayor que el máximo da 404 sin consultar la
  base.
- `confirmar` con otro valor ("SI", "1", "no", vacío) se trata como ausente: en un Sprint en curso se muestra la
  advertencia; en uno finalizado no cambia nada.
- Un Sprint finalizado sin registro de cierre de HU-15 (datos anteriores o cargados a mano): la tabla dice "No hay
  historias registradas en este Sprint", el avance "0 de 0 historias completadas (No disponible)" y 0 Story Points;
  las métricas de HU-07 y los defectos se muestran igual.
- Defectos resueltos en el Sprint pero detectados antes, sin detectados en él: la lista queda vacía y el resumen dice
  "0 defectos detectados y 1 resuelto en este Sprint".
- Un defecto detectado en el Sprint y resuelto en un Sprint posterior figura en la lista como "Resuelto" con su Sprint
  de resolución, y no suma a los resueltos de este Sprint.
- Un defecto reabierto (HU-19) figura como "Abierto", sin Sprint de resolución.
- Todas las historias con horas estimadas: no se muestra el aviso de historias sin horas. Ninguna: "0 horas
  estimadas en total" y "N historias sin horas estimadas".
- Desviación negativa: "-8,50 %"; sin diferencia: "0,00 %", sin signo.
- Horas con decimales se muestran con coma y sin ceros sobrantes: 2,5 y 0,25.
- Un Sprint sin registros de esfuerzo: "0 horas registradas en total" y ningún integrante listado.
- Un integrante que registró horas en historias del Sprint figura aunque ya no pertenezca al proyecto.
- Una historia que volvió al Backlog y se replanificó en un Sprint posterior acumula en sus horas reales los registros
  de los dos Sprints (criterio de HU-07): el reporte del Sprint anterior cambia si se registran horas nuevas.
- Si la cantidad de historias o de defectos detectados que lee el reporte no coincide con la que informa HU-10 para
  un Sprint finalizado (datos que cambiaron entre una lectura y otra), se muestra la página de error en lugar de un
  PDF con números que no cierran.
- `HEAD` sobre la descarga responde los mismos encabezados que `GET`, sin cuerpo.

Reporte del proyecto:

- Un proyecto con solo un Sprint en curso: su fila dice "En curso: se calcula al cerrar el Sprint", "Velocidad
  acumulada: 0 Story Points" y "Promedio: No disponible".
- Un proyecto con historias y sin Sprints: "Aún no hay Sprints", con el avance y las horas de sus historias.
- Una fila de un Sprint finalizado sin registro de cierre: "0 de 0" historias y "0 de 0" Story Points, con la
  velocidad y la desviación de HU-07.
- Desviación "No disponible" en una fila: se muestra tal cual, con " (cálculo parcial)" si HU-07 lo marca.
- Defectos en singular: "1 defecto registrado: 1 abierto y 0 resueltos" o "1 defecto registrado: 0 abiertos y 1
  resuelto". Todos resueltos: "2 defectos registrados: 0 abiertos y 2 resueltos" y "No hay defectos abiertos".
- Un proyecto que todavía no empezó o que ya terminó muestra el estado que calcule HU-12 ("No iniciado" o
  "Finalizado") con la fecha de generación.
- Un proyecto sin integrantes muestra "Integrantes: Sin integrantes".
- Las historias que nunca entraron a un Sprint cuentan en el avance y en las horas del proyecto.
- Muchos Sprints o defectos abiertos: las tablas pasan de página y repiten su encabezado.
- `confirmar` con otro valor se trata como ausente, igual que en el Sprint.
- Si el proyecto del contexto deja de existir mientras se arma el reporte, se muestra la página 404 de HU-20a, sin
  archivo.

Comunes:

- Títulos, descripciones u objetivos largos se cortan en varias líneas dentro de su columna; nunca se salen de la
  página. Una palabra más ancha que la columna se corta por caracteres.
- Caracteres que las fuentes estándar no pueden mostrar (emojis, letras de otros alfabetos) se reemplazan por "?";
  los acentos, la "ñ", "«»", "–", "—", comillas tipográficas, "…" y "€" se muestran bien. Los saltos de línea y
  tabulaciones de un texto se muestran como espacios.
- Paréntesis y barras invertidas en los textos ("Login (v2)", "C:\temp") se muestran tal cual.
- Muchas filas: el PDF pasa a varias páginas, repite el encabezado de la tabla en cada una y numera todas ("Página 1
  de 4" … "Página 4 de 4").
- Recargar o volver a exportar lee los datos del momento.

## Requisitos funcionales

Pantalla y reporte de un Sprint:

- **FR-001**: El sistema MUST ofrecer la sección "Reportes" en `/reportes`, que exige proyecto activo, y MUST
  responder solo a `GET` (y `HEAD`, que el enrutador de HU-20a atiende igual) en `/reportes`,
  `/reportes/sprints/{id}/pdf` y `/reportes/proyecto/pdf`; cualquier otro método responde 405.
- **FR-002**: La pantalla MUST listar los Sprints del proyecto activo, el activo primero y después los finalizados
  del número mayor al menor, con número, objetivo, estado y el enlace "Exportar PDF".
- **FR-003**: Sin Sprints, la pantalla MUST mostrar "Aún no hay Sprints para exportar", con respuesta 200.
- **FR-004**: Un `id` inválido, o de un Sprint inexistente o de otro proyecto, MUST responder la página 404 de HU-20a
  sin leer los datos del reporte ni generar archivo.
- **FR-005**: Para un Sprint finalizado, el sistema MUST responder el PDF como descarga, con el nombre
  `reporte-sprint-N.pdf`, el tipo de contenido de PDF y sin almacenarse en la caché.
- **FR-006**: Para un Sprint en curso sin `confirmar=si`, el sistema MUST mostrar la advertencia "El Sprint N sigue en
  curso: el reporte puede estar incompleto" con "Generar igual" y "Cancelar", y MUST NOT generar el PDF.
- **FR-007**: "Generar igual" MUST descargar el PDF marcado "Reporte parcial: Sprint en curso"; "Cancelar" MUST volver
  a "Reportes" sin generar nada. Cualquier valor de `confirmar` distinto de "si" MUST tratarse como ausente.
- **FR-008**: El encabezado del PDF de un Sprint MUST mostrar el proyecto, "Sprint N", el objetivo, las fechas en
  dd/mm/aaaa, el estado y la fecha y hora de generación.
- **FR-009**: La parte de historias MUST listar cada historia con título, estado, Story Points, horas estimadas y
  horas reales según R2 y R3, con "Sin estimar" donde falte un dato, y debajo "X de Y historias completadas (P %)"
  según R6 (en un Sprint en curso, "Se calculan al cerrar el Sprint", R9).
- **FR-010**: La parte de estimaciones MUST mostrar los Story Points planificados y completados (R6; en un Sprint en
  curso, "Se calculan al cerrar el Sprint", R9) y las horas estimadas totales con el aviso de historias sin horas
  estimadas (R4).
- **FR-011**: La parte de esfuerzo MUST mostrar las horas registradas en total y por integrante, ordenadas por
  nombre (R5).
- **FR-012**: La parte de métricas MUST mostrar la velocidad, la desviación y, si corresponde, el aviso parcial de
  HU-07 (R7); en un Sprint en curso MUST mostrar "Se calculan al cerrar el Sprint".
- **FR-013**: La parte de defectos MUST listar los detectados en el Sprint con descripción, severidad, estado,
  historia y Sprint de resolución si tiene, y MUST mostrar el resumen de R8 (o el de R9 en un Sprint en curso).
- **FR-015**: Los datos del reporte de un Sprint MUST salir de las fuentes de R2 a R9 sin recalcularse: HU-07 para
  velocidad, desviación y aviso; HU-10 para avance, Story Points y conteos de defectos; y una sola lectura consistente
  para lo demás.

Reporte del proyecto:

- **FR-022**: El sistema MUST ofrecer `GET /reportes/proyecto/pdf`, con el proyecto del contexto (nunca de la
  dirección), que descarga `reporte-proyecto-N.pdf` (N = identificador del proyecto) con los mismos encabezados que el
  reporte de un Sprint.
- **FR-023**: La pantalla "Reportes" MUST mostrar siempre el botón "Exportar reporte del proyecto", también sin
  Sprints.
- **FR-024**: Con un Sprint en curso y sin `confirmar=si`, el sistema MUST mostrar "El proyecto tiene un Sprint en
  curso (Sprint N): el reporte puede estar incompleto" con "Generar igual" y "Cancelar" y MUST NOT generar el PDF; con
  la confirmación, el PDF MUST llevar "Reporte parcial: hay un Sprint en curso". Sin Sprint en curso, MUST generarse
  directo, sin advertencia.
- **FR-025**: El encabezado del PDF del proyecto MUST mostrar el nombre, las fechas en dd/mm/aaaa, el estado de HU-12
  (R13), los integrantes en orden alfabético y la fecha y hora de generación.
- **FR-026**: La parte de avance MUST mostrar las historias por estado con ceros y "X de Y historias completadas
  (P %)" según R14.
- **FR-027**: La parte de Sprints MUST mostrar una fila por Sprint según R15 y debajo la velocidad acumulada y el
  promedio de R16; sin Sprints, "Aún no hay Sprints".
- **FR-028**: La parte de esfuerzo MUST mostrar las horas estimadas totales con el aviso de historias sin horas, las
  horas registradas totales y por integrante, según R17.
- **FR-029**: La parte de defectos MUST mostrar el resumen y la lista de abiertos de R18, o "No se registraron
  defectos en este proyecto".
- **FR-030**: Los datos del reporte del proyecto MUST salir sin recalcularse: HU-12 para estado y avance, HU-10 y
  HU-07 para cada Sprint finalizado, HU-08 para la acumulada y el promedio, y una sola lectura consistente del mismo
  puerto para lo demás.

Comunes:

- **FR-014**: Todas las páginas de cada PDF MUST tener el pie "Página X de Y", con Y igual a la cantidad real de
  páginas.
- **FR-016**: Todos los datos de los reportes MUST pertenecer al proyecto activo.
- **FR-017**: Cada PDF MUST armarse completo antes de enviarse; ante cualquier falla (lectura, HU-07, HU-08, HU-10,
  HU-12 o armado) MUST mostrarse la página de error de HU-20a (500, o 503 si la base no responde) y MUST NOT enviarse
  bytes del PDF.
- **FR-018**: Cada PDF MUST ser un documento válido (versión 1.4, tamaño A4, fuentes estándar) que muestra bien los
  acentos y la "ñ", corta los textos largos en varias líneas, pasa de página cuando hace falta y repite el
  encabezado de cada tabla en cada página.
- **FR-019**: El sistema MUST NOT agregar migraciones ni guardar los PDF ni datos propios; la única escritura es la
  que ya hace HU-07 al calcular, con los mismos valores.
- **FR-020**: Sin proyecto activo MUST aplicarse la redirección de HU-20b, con regreso a la dirección pedida
  ("Reportes" o una descarga).
- **FR-021**: Los textos de la pantalla y de los PDF MUST definirse en un único lugar cada uno; las respuestas de error
  y los PDF MUST NOT incluir texto de consultas ni detalles internos.

## Entidades clave

- **Proyecto** (HU-01, HU-12): nombre, fechas, estado por fechas e integrantes.
- **Sprint** (HU-04): número, objetivo, fechas, estado ("Activo" o "Finalizado") y proyecto.
- **Historia** (HU-02, HU-17): título, estado, Story Points y horas estimadas.
- **Registro de cierre** (HU-15): por cada historia que tenía el Sprint al cerrar, su estado final y sus Story Points.
- **Registro de esfuerzo** (HU-03): historia, integrante y horas.
- **Integrante**: nombre visible.
- **Defecto** (HU-06, HU-19): descripción, severidad, estado, historia, Sprint de detección y de resolución.
- **Métricas del Sprint** (HU-07): velocidad, desviación y aviso parcial.
- **Calidad y avance del Sprint** (HU-10): historias planificadas y completadas, porcentaje, Story Points
  planificados y completados, defectos detectados y resueltos.
- **Resumen de velocidad** (HU-08): velocidad acumulada y promedio de los Sprints finalizados.
- **Reporte del Sprint** (nuevo, no se guarda): las siete partes ya armadas con sus textos.
- **Reporte del proyecto** (nuevo, no se guarda): las seis partes ya armadas con sus textos.
- **Documento PDF** (nuevo, no se guarda): páginas con texto y tablas, listas para descargar.

## Criterios de éxito

- **SC-001**: En el 100% de las pruebas, los números de los dos PDF (historias, porcentajes, Story Points, horas,
  horas por integrante, velocidad, acumulada, promedio, desviación y defectos) coinciden con las cuentas a mano de
  "Datos de ejemplo".
- **SC-002**: En el 100% de las pruebas con Sprints finalizados, la velocidad y la desviación coinciden con las de
  HU-07, el avance, los Story Points y los conteos de defectos con los de HU-10, la acumulada y el promedio con los de
  HU-08, y el estado y el avance del proyecto con los de HU-12.
- **SC-003**: Ningún dato de otro proyecto aparece en la lista ni en los PDF.
- **SC-004**: Ninguna falla entrega un archivo: en el 100% de las pruebas de error la respuesta es la página de error
  y no contiene bytes de PDF.
- **SC-005**: El Scrum Master obtiene el PDF de un Sprint finalizado o el del proyecto (sin Sprint en curso) en dos
  acciones (abrir "Reportes" y presionar el enlace o el botón), y cada reporte de un proyecto con 120 historias y 50
  defectos se genera en menos de 1 s con la base local.
- **SC-006**: Los PDF de ejemplo (de un Sprint y del proyecto) se abren sin errores en dos lectores distintos (un
  navegador y un lector de escritorio), con acentos, saltos de página y "Página X de Y" correctos.
- **SC-007**: Cada bloque tiene pruebas que fallaron antes del código (evidencia RED) y pasan después (evidencia
  GREEN), y la cobertura de los paquetes nuevos queda en `cobertura.txt` con al menos 80%.

## Supuestos y decisiones

- Los nombres de HU-06 (`defects`), HU-08 (`SummarizeVelocity`), HU-10 (`QualityService`, `SprintQuality`), HU-15
  (`sprint_story_results`) y HU-17 (horas estimadas) salen de sus documentos (`specs/006-defects`,
  `specs/008-dashboard`, `specs/010-quality-metrics`, `specs/015-sprint-close`, `specs/017-story-estimates`), porque
  todavía no están en el código de `main`; se confirman en T001.
- HU-12 no tiene documentos en `specs/` ni código en `main`. De ella se toma solo el comportamiento que fija el Issue
  y que repiten otras specs (`001-fix-project-dates`, `021-web-projects`): estado "No iniciado", "En curso" o
  "Finalizado" calculado con las fechas (el primer y el último día son "En curso"), y avance "X de Y" con "0,00 %"
  sin historias. Los nombres de sus funciones en `src/project` se confirman en T001.
- La DoD de la entrega final de "Planificación de proyecto.pdf" pide el informe del proyecto en PDF; por eso la
  historia pasa a Should Have, con valor 70 y 8 SP. El documento no está en el repositorio: el resto de la
  especificación sale del Issue y de las notas del equipo.
- El avance del proyecto (HU-12) muestra "0,00 %" sin historias, y el de un Sprint (HU-10) muestra "No disponible"
  sin historias: cada reporte usa la regla de la historia de la que sale el dato.
- La velocidad de cada fila del proyecto es la de HU-07, y la acumulada y el promedio se calculan con la regla de
  HU-08 sobre esas mismas velocidades, para que la tabla y el resumen coincidan. HU-08 usa en su pantalla los Story
  Points completados de HU-10; los dos valores coinciden (HU-10, R6).
- La columna `user_stories.estimated_hours` ya está en `main` (migración 002); HU-17 agrega la pantalla para
  cargarla y bloquea su edición en historias de Sprints finalizados. Las historias que volvieron al Backlog pueden
  cambiar sus horas estimadas después del cierre, y los reportes muestran las actuales.
- HU-07 calcula con las historias "Hecho" que quedan asignadas al Sprint (las demás vuelven al Backlog al cerrar) y
  con la suma de todos los registros de esfuerzo de cada historia; HU-09 usa el mismo criterio para las horas
  reales, para que la tabla coincida con la desviación.
- HU-07 vuelve a escribir los mismos valores en la tabla de métricas de cada Sprint finalizado cada vez que se exporta
  un reporte, igual que en HU-25 y HU-08.
- La fecha y hora de generación es la del servidor, en formato dd/mm/aaaa hh:mm; la misma fecha se usa para el estado
  de HU-12.
- El estado del Sprint se muestra como "Finalizado" o "En curso".
- "Más reciente" es el Sprint de mayor número.
- El Issue nombra todas las historias, fechas y defectos de «Alfa» y «Beta»; las fechas de «Beta» y su integrante
  solo se fijan para las pruebas. Las pruebas suman proyectos propios que no cambian las cuentas de «Alfa»: «Delta»
  (el Sprint de otro proyecto del escenario 12 y un integrante, «Caro», que ya no está en el proyecto), «Gamma»
  (defectos resueltos en otro Sprint, Sprint sin registro de cierre y proyecto sin Sprint en curso) y «Largo»
  (rendimiento y saltos de página); ver research.md §15.

## Trazabilidad

| Escenario | Requisitos | Éxito | Prueba de pantalla / BDD | Prueba de dominio / integración |
|---|---|---|---|---|
| 1 Exportación exitosa | FR-005, FR-008, FR-014 | SC-005 | `TestHU09_ExportarFinalizado`, `TestHU09BDD_ExportacionExitosaDeUnSprintFinalizado` (BDD "Exportación exitosa de un Sprint finalizado") | `TestBuildSprintReport`, `TestRenderSprintReport` |
| 2 Historias | FR-009, FR-015 | SC-001, SC-002 | `TestHU09_ContenidoDelPDF`, `TestHU09BDD_HistoriasPlanificadasYCompletadas` (BDD "Historias planificadas y completadas") | `TestBuildSprintReport`, `TestPostgresReportStore_Finalizado` |
| 3 Estimaciones | FR-009, FR-010 | SC-001 | `TestHU09_ContenidoDelPDF` | `TestBuildSprintReport`, `TestReportTexts` |
| 4 Esfuerzo | FR-011 | SC-001 | `TestHU09_ContenidoDelPDF` | `TestPostgresReportStore_Finalizado`, `TestReportTexts` |
| 5 Métricas | FR-012, FR-015 | SC-001, SC-002 | `TestHU09_ContenidoDelPDF`, `TestHU09BDD_MetricasDelSprint` (BDD "Métricas del Sprint") | `TestBuildSprintReport`, `TestReport_CoherenciaConHU07YHU10` |
| 6 Defectos | FR-013, FR-015 | SC-001, SC-002 | `TestHU09_ContenidoDelPDF` | `TestPostgresReportStore_Finalizado`, `TestReportTexts` |
| 7 Sin defectos | FR-013 | SC-001 | `TestHU09_SinDefectos` | `TestReportTexts` |
| 8 Advertencia | FR-006 | — | `TestHU09_AdvertenciaSprintEnCurso`, `TestHU09BDD_AdvertenciaAlExportarUnSprintEnCurso` (BDD "Advertencia al exportar un Sprint en curso") | — |
| 9 Generar igual | FR-007, FR-009, FR-010, FR-012, FR-013 | SC-001 | `TestHU09_GenerarIgual` | `TestBuildSprintReport`, `TestPostgresReportStore_EnCurso` |
| 10 Cancelar | FR-007 | — | `TestHU09_AdvertenciaSprintEnCurso` | — |
| 11 Sin Sprints | FR-002, FR-003, FR-023 | — | `TestHU09_ListaDeSprints`, `TestHU09BDD_SinSprints` (BDD "Sin Sprints") | `TestPostgresReportCatalog` |
| 12 Inexistente o ajeno | FR-004, FR-016 | SC-003 | `TestHU09_SprintInexistenteOAjeno` | `TestPostgresReportCatalog`, `TestPostgresReportStore_OtroProyecto` |
| 13 Error al generar | FR-017, FR-021 | SC-004 | `TestHU09_ErrorAlGenerar`, `TestHU09_ProyectoErrores` | `TestReportService_Errores`, `TestProjectReportService_Errores` |
| 14 Advertencia del proyecto | FR-024 | — | `TestHU09_ProyectoAdvertencia`, `TestHU09BDD_AdvertenciaAlExportarUnProyectoConUnSprintEnCurso` (BDD "Advertencia al exportar un proyecto con un Sprint en curso") | — |
| 15 Encabezado del proyecto | FR-022, FR-024, FR-025 | SC-005 | `TestHU09_ProyectoContenido` | `TestBuildProjectReport`, `TestRenderProjectReport` |
| 16 Avance y Sprints | FR-026, FR-027, FR-030 | SC-001, SC-002 | `TestHU09_ProyectoContenido`, `TestHU09BDD_AvanceYSprintsDelProyecto` (BDD "Avance y Sprints del proyecto") | `TestBuildProjectReport`, `TestPostgresReportStore_Proyecto`, `TestProjectReport_Coherencia` |
| 17 Esfuerzo y defectos | FR-028, FR-029 | SC-001 | `TestHU09_ProyectoContenido` | `TestPostgresReportStore_Proyecto`, `TestProjectReportTexts` |
| 18 Proyecto sin Sprints ni historias | FR-023, FR-024, FR-026, FR-027, FR-028, FR-029 | SC-001 | `TestHU09_ProyectoSinSprints`, `TestHU09BDD_ProyectoSinSprintsNiHistorias` (BDD "Proyecto sin Sprints ni historias") | `TestPostgresReportStore_Proyecto`, `TestRenderProjectReport` |
| PDF válido | FR-014, FR-018 | SC-006 | Verificación manual (T049) | `TestPDF_Estructura`, `TestPDF_Texto`, `TestPDF_Anchos`, `TestPDF_CorteDeLineas`, `TestPDF_SaltoDePagina` |
| (Caso límite) Resueltos detectados antes | FR-013 | SC-001 | `TestHU09_ResueltosDetectadosAntes` | `TestPostgresReportStore_Finalizado`, `TestReportTexts` |
| (Caso límite) Sprint sin registro de cierre | FR-009, FR-010 | SC-001 | `TestHU09_SprintSinRegistroDeCierre` | `TestPostgresReportStore_Finalizado`, `TestRenderSprintReport` |
| (Caso límite) `id` y `confirmar` | FR-004, FR-007 | — | `TestHU09_SprintInexistenteOAjeno`, `TestHU09_AdvertenciaSprintEnCurso` | `TestReportView` |
| (Caso límite) Proyecto sin Sprint en curso | FR-024, FR-027 | — | `TestHU09_ProyectoSinSprintActivo` | `TestRenderProjectReport` |
| (HU-20b) Sin proyecto | FR-020 | — | `TestHU09_SinProyectoYMetodo`, `TestHU09_ProyectoErrores` | `TestRegisterReports_ArgumentosNil` |
| (HU-20a) Solo lectura | FR-001 | — | `TestHU09_SinProyectoYMetodo` | — |
| Rendimiento | — | SC-005 | `TestHU09_Rendimiento` | `TestPDF_SaltoDePagina` |
| Sin persistencia propia | FR-019 | — | `TestHU09_AdvertenciaSprintEnCurso` (sin filas en `sprint_metrics`) | Revisión en T047 (`git diff --stat main`) |
| Aislamiento entre proyectos | FR-016 | SC-003 | `TestHU09_SprintInexistenteOAjeno`, `TestHU09_ProyectoContenido` | `TestPostgresReportStore_OtroProyecto`, `TestPostgresReportStore_Proyecto` |

### Dónde está cada prueba (T050)

| Archivo | Pruebas |
|---|---|
| `src/pdf/winansi_test.go`, `widths_test.go` | `TestPDF_Texto`, `TestPDF_Anchos`, `TestPDF_CorteDeLineas` |
| `src/pdf/document_test.go` (con el lector de `pdftest_test.go`) | `TestPDF_Estructura`, `TestPDF_SaltoDePagina` |
| `src/report/texts_test.go` | `TestReportTexts`, `TestProjectReportTexts` |
| `src/report/service_test.go`, `project_service_test.go` | `TestBuildSprintReport`, `TestReportService_Errores`, `TestBuildProjectReport`, `TestProjectReportService_Errores` |
| `src/report/document_test.go`, `project_document_test.go` | `TestRenderSprintReport`, `TestRenderProjectReport` |
| `src/report/postgres_store_integration_test.go` (datos de `fixtures_test.go`) | `TestPostgresReportStore_Finalizado`, `_EnCurso`, `_OtroProyecto`, `_Proyecto`, `TestReport_CoherenciaConHU07YHU10`, `TestProjectReport_Coherencia` |
| `src/web/postgres_report_catalog_integration_test.go` (datos de `report_fixtures_test.go`) | `TestPostgresReportCatalog` |
| `src/web/report_view_test.go` | `TestReportView` |
| `src/web/report_handlers_test.go` | `TestHU09_ListaDeSprints`, `_ExportarFinalizado`, `_ContenidoDelPDF`, `_SinDefectos`, `_ResueltosDetectadosAntes`, `_SprintSinRegistroDeCierre`, `_AdvertenciaSprintEnCurso`, `_GenerarIgual`, `_SprintInexistenteOAjeno`, `_ErrorAlGenerar`, `_SinProyectoYMetodo`, `_Rendimiento`, `TestRegisterReports_ArgumentosNil` |
| `src/web/report_project_handlers_test.go` | `TestHU09_ProyectoAdvertencia`, `_ProyectoContenido`, `_ProyectoSinSprints`, `_ProyectoSinSprintActivo`, `_ProyectoErrores` |
| `test/web_report_bdd_test.go` | BDD: `TestHU09BDD_ExportacionExitosaDeUnSprintFinalizado`, `_HistoriasPlanificadasYCompletadas`, `_MetricasDelSprint`, `_AdvertenciaAlExportarUnSprintEnCurso`, `_SinSprints`, `_AdvertenciaAlExportarUnProyectoConUnSprintEnCurso`, `_AvanceYSprintsDelProyecto`, `_ProyectoSinSprintsNiHistorias` |

Código: `src/pdf` (generador), `src/report` (modelo, textos, servicio, documentos y `PostgreSQLStore`) y, en
`src/web`, `report_catalog.go`, `postgres_report_catalog.go`, `report_view.go`, `report_handlers.go` y las plantillas
`reports.html` y `report_warning.html`; registro en `src/cmd/server/main.go`.

Nota de implementación: un Sprint finalizado sin horas estimadas tiene en HU-07 la desviación "No disponible" marcada
como cálculo parcial, así que su fila en el reporte del proyecto muestra "No disponible (cálculo parcial)"
(`TestHU09_ProyectoSinSprintActivo`, Sprints 1 y 2 de «Gamma»).
