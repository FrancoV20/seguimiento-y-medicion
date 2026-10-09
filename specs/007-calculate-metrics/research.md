# Research: Calcular métricas del Sprint

## Decision 1: Servicio de dominio Go con persistencia PostgreSQL

- **Decision**: Implementar el cálculo en un servicio Go separado de `PostgreSQLMetricsStore`, que usará `github.com/jackc/pgx/v5` mediante `database/sql` para cargar datos y persistir el resultado.
- **Rationale**: La constitución v1.1.0 exige PostgreSQL y una implementación concreta para toda interfaz de almacenamiento antes de integrar la feature.
- **Alternatives considered**: Calcular solo en memoria o dejar la persistencia para después. Se descartan porque no cumplen el principio VI.

## Decision 2: Velocidad basada en historias terminadas

- **Decision**: La velocidad será la suma de Story Points de las historias terminadas del Sprint.
- **Rationale**: Coincide con FR-002 y evita contar trabajo pendiente como completado.
- **Alternatives considered**: Sumar todas las historias asignadas. Se descarta porque sobrestima el trabajo entregado.

## Decision 3: Desviación porcentual firmada

- **Decision**: Calcular `((real - estimado) / estimado) * 100` y redondear a dos posiciones decimales.
- **Rationale**: El signo distingue sobre-ejecución positiva de subejecución negativa y la fórmula está fijada en FR-005.
- **Alternatives considered**: Usar valor absoluto. Se descarta porque pierde la dirección de la desviación.

## Decision 4: Cálculo parcial explícito

- **Decision**: Si algunas historias terminadas carecen de horas estimadas o reales completas, calcular con las completas y mostrar `Cálculo parcial: basado en X de Y historias`.
- **Rationale**: Permite información útil sin imputar valores; `Y` es el total de historias terminadas y `X` las usadas.
- **Alternatives considered**: Rechazar todo el cálculo o imputar cero. Se descartan porque contradicen la decisión del equipo o generan métricas engañosas.

## Decision 5: Datos inválidos bloquean el cierre

- **Decision**: Validar horas reales negativas y Story Points inválidos antes de cerrar; devolver todas las historias afectadas y sus motivos.
- **Rationale**: Los datos inválidos deben corregirse antes de producir métricas definitivas.
- **Alternatives considered**: Excluir la historia inválida y cerrar. Se descarta porque el equipo definió un bloqueo duro.

## Decision 6: Sin denominador, resultado no disponible

- **Decision**: Si el esfuerzo estimado total es cero o no existe, devolver `No disponible` sin dividir.
- **Rationale**: Evita división por cero, infinito y valores no numéricos.
- **Alternatives considered**: Devolver cero. Se descarta porque ausencia de base no equivale a desviación cero.

## Decision 7: Adaptador PostgreSQL antes de integrar

- **Decision**: `PostgreSQLMetricsStore` debe implementar la carga de fuentes y persistencia de métricas mediante `database/sql` y pgx, con pruebas de integración PostgreSQL, antes de integrar el servicio.
- **Rationale**: Cumple la obligación constitucional de contar con implementación concreta contra PostgreSQL.
- **Alternatives considered**: Integrar solo un store en memoria o mocks. Se descartan porque no verifican el motor real.

## Decision 8: Pruebas unitarias y de integración

- **Decision**: Usar `go test ./...`, `testing` y tablas de casos para el cálculo; probar transacciones, lectura de fuentes y persistencia de resultados contra PostgreSQL.
- **Rationale**: Se mantiene el ciclo TDD rápido y se verifica el contrato real de almacenamiento.
- **Alternatives considered**: Solo pruebas unitarias. Se descartan porque dejarían sin validar `PostgreSQLMetricsStore`.
