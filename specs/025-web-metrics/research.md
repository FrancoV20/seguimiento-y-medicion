# Research: Interfaz web de métricas del Sprint

## Decision 1: Pantalla de solo lectura con lista de Sprints finalizados

- **Decision**: Una sección `/metricas` con la lista de Sprints finalizados del proyecto (más reciente primero) y las
  métricas del elegido con `?sprint=ID`; por defecto, el más reciente.
- **Rationale**: "Cuando abro sus métricas" necesita una forma de elegir el Sprint. Un enlace por Sprint funciona sin
  JavaScript y no requiere formularios, así que la pantalla no necesita protección de envíos.
- **Alternatives considered**: Mostrar solo el último Sprint (descartada: no permite comparar iteraciones); una
  pantalla por Sprint dentro de HU-16 (descartada: HU-16 es la consulta de Sprints anteriores y no es parte del MVP).

## Decision 2: La web no recalcula las métricas

- **Decision**: El handler pide a HU-07, mediante `MetricsSource`, la velocidad, la desviación y los valores X e Y del
  cálculo parcial, y solo los formatea.
- **Rationale**: HU-07 es la única fuente de la regla; recalcular en la web las desincronizaría. Es el mismo criterio
  que en HU-24 con las reglas de HU-03.
- **Alternatives considered**: Calcular con SQL en el catálogo (descartada: duplica la regla).

## Decision 3: Se quita el escenario "Cierre bloqueado"

- **Decision**: HU-25 no muestra cierres bloqueados; la lista de historias a corregir se muestra en "Cierre de
  Sprint" (HU-15).
- **Rationale**: El cierre es una acción de HU-15 y su resultado se informa donde se hace. Esta pantalla solo muestra
  Sprints cerrados, cuyos datos ya no cambian (HU-03, HU-17 y HU-13 bloquean cambios).
- **Alternatives considered**: Mostrar los Sprints bloqueados como "no se pudo cerrar" (descartada: un Sprint que
  no se cerró sigue activo y no tiene métricas definitivas).

## Decision 4: Solo Sprints finalizados

- **Decision**: La lista incluye únicamente Sprints con estado "Finalizado"; el Sprint activo se omite y se informa
  que sus métricas se calculan al cerrarlo.
- **Rationale**: El criterio dice "Sprint finalizado" y la descripción de HU-15, que cerrar "habilita el cálculo de
  métricas". Mostrar métricas a medias de un Sprint en curso confundiría la velocidad.
- **Alternatives considered**: Incluir el Sprint activo con métricas provisionales (descartada: no lo pide ninguna
  historia y rompe la definición de velocidad).

## Decision 5: Formato de la desviación

- **Decision**: Dos decimales, coma y signo (`+20,00 %`, `-8,50 %`, `0,00 %`), con la fórmula de HU-07 como ayuda
  ((horas reales − horas estimadas) ÷ horas estimadas) y la aclaración de que positivo significa que se trabajó más
  de lo estimado.
- **Rationale**: El signo hace visible si el equipo se pasó o se quedó corto. Un formato fijo permite probar el valor
  exacto.
- **Alternatives considered**: Un decimal o sin signo (descartada: pierde la dirección del desvío).

## Decision 6: Textos de HU-07 reutilizados

- **Decision**: "No disponible" y "Cálculo parcial: basado en X de Y historias" se toman de HU-07 si los expone; si
  no, se usan los textos del criterio con los valores X e Y de HU-07 (T001).
- **Rationale**: El criterio los fija literalmente y HU-07 ya los implementa.

## Decision 7: Un `sprint` inválido se ignora

- **Decision**: Un `sprint` no numérico, inexistente, activo o de otro proyecto se trata como si no estuviera.
- **Rationale**: Es una consulta, no una acción; un enlace viejo no debe producir una pantalla de error. No revela si
  el Sprint existe en otro proyecto.

## Decision 8: Catálogo de solo lectura propio

- **Decision**: `MetricsCatalog` con `FinishedSprints` y `ActiveSprintNumber`, y su adaptador
  `PostgreSQLMetricsCatalog` en `src/web`.
- **Rationale**: Mantiene la lista de Sprints separada del cálculo y evita editar archivos de HU-04, HU-15 y HU-23.

## Decision 9: Sin integrante requerido

- **Decision**: La sección requiere proyecto pero no integrante.
- **Rationale**: Es una consulta sin acciones a nombre de nadie; no hay motivo para exigir identificación.

## Decision 10: Estrategia de pruebas

- **Decision**: Pruebas de handler y de integración contra PostgreSQL real con fixtures propias (dos proyectos; Sprints
  finalizados con historias "Hecho" y horas estimadas y reales conocidas; uno sin horas estimadas; uno con datos
  parciales; uno sin historias completadas cerrado por el servicio de HU-15; un Sprint activo); el servicio de HU-07
  real; un catálogo o una fuente que fallan solo para simular las fallas de lectura y de cálculo; BDD.
- **Nota de fixtures**: `user_stories.project_id` no tiene borrado en cascada: se borran primero las historias y
  después el proyecto.