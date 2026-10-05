# Quickstart: Calcular métricas del Sprint

## Prerequisites

- Go 1.27.1 o posterior instalado.
- Docker Desktop instalado y en ejecución.
- PostgreSQL levantado con Docker Compose para las pruebas de integración.
- Herramienta `golang-migrate/migrate` instalada.
- Paquete `src/metrics`, `PostgreSQLMetricsStore` y `go.mod` creados según el plan.
- Variable `DATABASE_URL` configurada para PostgreSQL, por ejemplo:
   `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.

## Run the tests

Desde la raíz del repositorio, iniciar PostgreSQL:

```powershell
docker compose up -d postgres
```

Aplicar las migraciones pendientes en orden numérico. La migración 004 depende de que las
migraciones 001, que crea los proyectos, 002, que crea las historias del backlog, y 003, que
crea los Sprints, ya estén aplicadas porque las métricas leen Sprints e historias existentes:

```powershell
migrate -path db/migrations -database "$DATABASE_URL" up
```

Luego ejecutar las pruebas:

```powershell
go test ./...
```

El comando debe finalizar correctamente y ejecutar las pruebas unitarias de métricas, los
escenarios BDD y la integración de `PostgreSQLMetricsStore` contra PostgreSQL.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Sprint finalizado con datos completos:
   - velocidad igual a la suma de Story Points terminados;
   - desviación con `((real - estimado) / estimado) * 100`;
   - redondeo a dos decimales y signo correcto.
2. Sin esfuerzo estimado utilizable:
   - desviación `No disponible`;
   - no hay división por cero, infinito ni valor no numérico.
3. Datos incompletos:
   - cálculo solo con historias completas;
   - alerta exacta `Cálculo parcial: basado en X de Y historias`.
4. Datos inválidos antes del cierre:
   - horas reales negativas bloquean el cierre;
   - Story Points inválidos bloquean el cierre;
   - se identifican todas las historias y campos a corregir.
5. Sin historias terminadas:
   - velocidad `0`;
   - desviación `No disponible`.
6. PostgreSQL:
   - `PostgreSQLMetricsStore` usa pgx mediante `database/sql`;
   - carga fuentes y persiste el snapshot de métricas en PostgreSQL;
   - una falla transaccional no deja snapshots parciales;
   - la prueba de integración se ejecuta contra PostgreSQL real.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio.
Las pruebas deben ejecutar las reglas reales de cálculo y persistencia, no reemplazarlas con
mocks o métodos exclusivos de test.
