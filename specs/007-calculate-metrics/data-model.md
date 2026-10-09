# Data Model: Calcular métricas del Sprint

## Sprint

Iteración cuyos datos se validan y consolidan.

### Fields

- `id`: identificador único del Sprint.
- `status`: debe ser `Finalizado` para exponer métricas definitivas.
- `stories`: historias asociadas.

### Rules

- El cierre debe bloquearse si alguna historia terminada tiene horas reales negativas o Story
  Points inválidos.
- Un Sprint sin historias terminadas puede mostrar velocidad 0 y desviación `No disponible`.

## UserStory

Historia terminada incluida en el Sprint.

### Fields

- `id`: identificador único.
- `status`: solo `Terminada` participa en estas métricas.
- `storyPoints`: estimación validada.
- `estimatedHours`: horas estimadas, pueden faltar para el cálculo parcial.
- `actualHours`: horas reales, deben ser no negativas.

### Rules

- Una historia no terminada se excluye de velocidad y esfuerzo.
- Story Points inválidos u horas reales negativas bloquean el cierre.
- Horas estimadas o reales ausentes no bloquean por sí solas, pero excluyen la historia de la
  desviación y activan el conteo parcial.

## SprintMetrics

Resultado calculado y persistido para un Sprint finalizado.

### Fields

- `sprintID`: Sprint al que pertenece.
- `velocity`: suma de Story Points terminados.
- `estimatedHoursTotal`: suma de horas estimadas usadas.
- `actualHoursTotal`: suma de horas reales usadas.
- `deviationPercentage`: porcentaje firmado a dos decimales o `No disponible`.
- `calculationUsedCount`: cantidad `X` de historias completas.
- `calculationTotalCount`: cantidad `Y` de historias terminadas.
- `isPartial`: verdadero cuando `X < Y` por datos faltantes.
- `warning`: alerta exacta del cálculo parcial, si corresponde.

### Rules

- `velocity = sum(storyPoints)` de historias terminadas.
- `deviation = ((actualHoursTotal - estimatedHoursTotal) / estimatedHoursTotal) * 100`.
- Si `estimatedHoursTotal == 0`, la desviación es `No disponible`.
- Si `isPartial`, la alerta es `Cálculo parcial: basado en X de Y historias`.

## MetricsStore

Interfaz interna para leer fuentes y persistir resultados de métricas.

### Operations

- `LoadSprintData(sprintID)`: carga Sprint e historias necesarias para validar y calcular.
- `SaveMetrics(metrics)`: persiste el resultado asociado al Sprint.

## PostgreSQLMetricsStore

Implementación concreta de `MetricsStore` contra PostgreSQL.

### Dependencies

- `github.com/jackc/pgx/v5` como driver.
- `github.com/jackc/pgx/v5/stdlib` para operar mediante `database/sql`.

### Rules

- Debe leer Sprint, historias y esfuerzos desde PostgreSQL.
- Debe persistir el snapshot de métricas mediante una transacción de `database/sql`.
- Debe contar con pruebas de integración contra PostgreSQL antes de integrarse.
