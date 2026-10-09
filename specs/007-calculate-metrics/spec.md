# Feature Specification: Calcular métricas del Sprint

**Feature Branch**: `004-calculate-metrics`

**Created**: 2026-09-30

**Status**: Draft

**Input**: User description: "HU-07 (Issue #9): Sistema para que se calcule automáticamente la velocidad del equipo y la desviación entre esfuerzo estimado y real de un sprint, con el objetivo de evaluar si el equipo está cumpliendo con lo planificado."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Consultar velocidad y desviación de esfuerzo (Priority: P1)

Como responsable del seguimiento del proyecto, quiero consultar automáticamente la velocidad
del equipo y la desviación entre el esfuerzo estimado y el real de un Sprint finalizado para
evaluar el cumplimiento de la planificación.

**Why this priority**: Estas métricas permiten comparar el compromiso asumido con el trabajo
completado y detectar diferencias que deben considerarse en futuras planificaciones.

**Independent Test**: Puede probarse con un Sprint finalizado que tenga historias terminadas,
Story Points, horas estimadas y horas reales cargadas, verificando que la velocidad y la
desviación se calculan y muestran con los valores esperados.

**Acceptance Scenarios**:

1. **Given** un Sprint que acaba de finalizar, **When** el sistema procesa las horas estimadas
   totales versus las horas reales cargadas en las historias terminadas, **Then** calcula y
   muestra el porcentaje de desviación de esfuerzo con precisión matemática.
2. **Given** un Sprint finalizado con historias terminadas, **When** el sistema procesa las
   historias y sus Story Points, **Then** calcula y muestra la velocidad como el total de
   Story Points completados en el Sprint.

---

### User Story 2 - Informar ausencia de esfuerzo estimado (Priority: P2)

Como responsable del seguimiento, quiero que el sistema indique cuando no hay esfuerzo
estimado disponible para calcular la desviación, evitando resultados engañosos o errores.

**Why this priority**: Un indicador inválido puede llevar a conclusiones incorrectas sobre
el cumplimiento del equipo; informar la ausencia de datos mantiene la confianza en la métrica.

**Independent Test**: Puede probarse con un Sprint finalizado cuyas historias no tengan
horas estimadas, verificando que la desviación se muestra como "No disponible" y que la
velocidad continúa calculándose si existen Story Points terminados.

**Acceptance Scenarios**:

1. **Given** un Sprint finalizado donde ninguna historia tiene horas estimadas cargadas,
   **When** el sistema intenta calcular el porcentaje de desviación, **Then** evita la
   división por cero y muestra el indicador como "No disponible" en lugar de un error o un
   valor inválido.
2. **Given** un Sprint finalizado donde solo algunas historias terminadas tienen horas
  estimadas cargadas, **When** el sistema calcula la desviación de esfuerzo, **Then** calcula
  el porcentaje usando únicamente las historias con datos completos y muestra una alerta con
  el texto "Cálculo parcial: basado en X de Y historias".

---

### User Story 3 - Validar datos antes de cerrar el Sprint (Priority: P2)

Como Scrum Master, quiero que el sistema valide los datos de las historias terminadas antes
de cerrar un Sprint para corregir inconsistencias que invalidarían sus métricas.

**Why this priority**: Impedir el cierre con horas reales negativas o Story Points inválidos
protege la confiabilidad de la velocidad y de la desviación de esfuerzo.

**Independent Test**: Puede probarse intentando cerrar un Sprint con historias terminadas que
tengan datos inválidos y verificando que el cierre se bloquea e identifica las historias que
deben corregirse.

**Acceptance Scenarios**:

1. **Given** un Sprint con historias terminadas que tienen horas reales negativas o Story
  Points inválidos, **When** el Scrum Master intenta cerrar el Sprint, **Then** el sistema
  bloquea el cierre y muestra un mensaje indicando qué historias tienen datos inválidos que
  deben corregirse.

---

### Edge Cases

- Si el Sprint no está finalizado, el sistema no debe presentar sus métricas como definitivas.
- Si no hay historias terminadas, la velocidad debe mostrarse como 0 Story Points y la
  desviación debe mostrarse como "No disponible" cuando no exista esfuerzo estimado utilizable.
- Si algunas historias terminadas tienen estimación y otras no, el cálculo de desviación debe
  usar únicamente las historias con horas estimadas y reales completas, y mostrar la alerta
  "Cálculo parcial: basado en X de Y historias".
- Si el esfuerzo estimado total es exactamente 0 horas, la desviación debe ser "No disponible"
  aunque existan horas reales.
- Si las horas reales son menores que las estimadas, la desviación debe conservar signo negativo
  para indicar una subejecución respecto de lo planificado.
- Si las horas reales son mayores que las estimadas, la desviación debe conservar signo positivo
  para indicar una sobre-ejecución respecto de lo planificado.
- Si una historia terminada tiene horas reales negativas o Story Points inválidos, el sistema
  debe bloquear el cierre del Sprint, identificar la historia y solicitar la corrección antes
  de permitir calcular sus métricas definitivas.
- Los datos de un Sprint finalizado deben producir el mismo resultado al volver a consultar las
  métricas mientras no cambien los datos fuente autorizados.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST calcular automáticamente las métricas de un Sprint cuando este
  pasa a estado finalizado.
- **FR-002**: El sistema MUST calcular la velocidad como la suma de los Story Points de las
  historias terminadas pertenecientes al Sprint.
- **FR-003**: El sistema MUST sumar las horas estimadas de las historias terminadas para obtener
  el esfuerzo estimado total del Sprint.
- **FR-004**: El sistema MUST sumar las horas reales cargadas en las historias terminadas para
  obtener el esfuerzo real total del Sprint.
- **FR-005**: El sistema MUST calcular la desviación porcentual firmada con la fórmula
  `((esfuerzo real total - esfuerzo estimado total) / esfuerzo estimado total) × 100`.
- **FR-006**: El sistema MUST mostrar la desviación con dos posiciones decimales, aplicando
  redondeo matemático consistente.
- **FR-007**: Cuando el esfuerzo estimado total sea 0 horas o no exista, el sistema MUST evitar
  la división por cero y mostrar la desviación como "No disponible".
- **FR-008**: El sistema MUST mostrar la velocidad, el esfuerzo estimado total, el esfuerzo real
  total y la desviación asociadas al Sprint finalizado.
- **FR-009**: El sistema MUST conservar el signo de la desviación: un resultado positivo indica
  que el esfuerzo real superó al estimado y un resultado negativo indica que fue menor.
- **FR-010**: El sistema MUST excluir del cálculo las historias que no estén terminadas y debe
  identificar las historias terminadas con datos incompletos para aplicar el cálculo parcial.
- **FR-011**: Cuando solo una parte de las historias terminadas tenga horas estimadas y reales
  completas, el sistema MUST calcular la desviación usando únicamente esas historias y MUST
  mostrar la alerta "Cálculo parcial: basado en X de Y historias", donde X es la cantidad
  utilizada y Y el total de historias terminadas.
- **FR-012**: Antes de cerrar un Sprint, el sistema MUST validar que las historias terminadas
  no tengan horas reales negativas ni Story Points inválidos.
- **FR-013**: Si alguna historia terminada tiene horas reales negativas o Story Points inválidos,
  el sistema MUST bloquear el cierre del Sprint, identificar las historias afectadas y solicitar
  la corrección de sus datos.
- **FR-014**: El sistema MUST indicar cuando no existen historias terminadas suficientes para
  calcular una métrica y no debe mostrar un valor numérico inventado.
- **FR-015**: El cálculo debe ser reproducible: con los mismos datos de Sprint, historias y
  esfuerzos, el sistema MUST producir los mismos resultados.

### Key Entities *(include if feature involves data)*

- **Sprint finalizado**: Iteración cuyo trabajo terminó y cuyos datos pueden utilizarse para
  consolidar métricas de desempeño.
- **Historia terminada**: Historia del Sprint que alcanzó el estado de finalización y cuyos
  Story Points, horas estimadas y horas reales pueden participar del cálculo.
- **Métrica de velocidad**: Total de Story Points completados en un Sprint finalizado.
- **Métrica de desviación de esfuerzo**: Diferencia porcentual firmada entre el esfuerzo real
  y el esfuerzo estimado del Sprint.
- **Esfuerzo**: Cantidad de horas estimadas o reales asociadas a las historias terminadas.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Para un Sprint finalizado con datos válidos, el 100% de los cálculos de desviación
  coincide con la fórmula definida, con una diferencia máxima de 0,01 puntos porcentuales por
  redondeo a dos decimales.
- **SC-002**: Para un Sprint finalizado, la velocidad mostrada coincide en el 100% de los casos
  con la suma de Story Points de sus historias terminadas.
- **SC-003**: El 100% de los Sprints sin esfuerzo estimado muestra "No disponible" y no muestra
  errores de división por cero, valores infinitos o valores no numéricos.
- **SC-004**: Al menos el 90% de los responsables de seguimiento de prueba puede interpretar
  correctamente la velocidad y la desviación mostradas.
- **SC-005**: El sistema muestra las métricas de un Sprint finalizado en menos de 2 segundos
  después de solicitar su consulta, bajo condiciones normales de uso.
- **SC-006**: El 100% de los cálculos parciales muestra el conteo exacto en el formato
  "Cálculo parcial: basado en X de Y historias".
- **SC-007**: El 100% de los intentos de cerrar un Sprint con horas reales negativas o Story
  Points inválidos es bloqueado y señala las historias que deben corregirse.

## Assumptions

- Solo las historias en estado "Terminada" participan en la velocidad y en los totales de
  esfuerzo, de acuerdo con el objetivo de medir el trabajo completado.
- Las horas estimadas y reales se expresan en la misma unidad: horas.
- La desviación es una métrica firmada: positiva significa sobre-ejecución y negativa significa
  que se emplearon menos horas que las estimadas.
- Las métricas se consultan para un Sprint finalizado y no sustituyen los datos originales de
  esfuerzo ni Story Points.
- Si faltan horas estimadas o reales en algunas historias terminadas, el sistema calcula con
  las historias que tienen datos completos y muestra el conteo de historias utilizadas sobre
  el total; no imputa valores faltantes.
- Las horas reales negativas y los Story Points inválidos son datos bloqueantes: deben corregirse
  antes de cerrar el Sprint y calcular sus métricas definitivas.
- La comparación histórica entre Sprints, gráficos y exportación de reportes quedan fuera del
  alcance de esta historia.
