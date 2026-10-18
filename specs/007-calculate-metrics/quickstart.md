# Quickstart: Calcular métricas del Sprint

## Prerequisites

- Go 1.27.1 instalado en Windows desde [go1.27.1.windows-amd64.msi](https://go.dev/dl/go1.27.1.windows-amd64.msi); verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- PostgreSQL levantado con Docker Compose para las pruebas de integración.
- Herramienta `golang-migrate/migrate` instalada; seguir la instalación de Windows documentada en [README.md](../../README.md).
- Paquete `src/metrics`, `PostgreSQLMetricsStore` y `go.mod` creados según el plan.
- Variable `DATABASE_URL` configurada con la cadena indicada en PostgreSQL local.

## PostgreSQL local

El servicio `postgres` de Docker Compose usa estos valores:

| Variable | Valor |
|---|---|
| Usuario | `sym_user` |
| Contraseña | `sym_pass` |
| Puerto del host | `5433` |
| Base de datos | `seguimiento_y_medicion` |

Desde PowerShell, configurá la conexión para las herramientas y pruebas:

```powershell
$env:DATABASE_URL =  "postgres://USUARIO:CLAVE@localhost:PUERTO/seguimiento_y_medicion?sslmode=disable""
```

Iniciá PostgreSQL y comprobá que acepta conexiones:

```powershell
docker compose up -d postgres
docker compose exec -T postgres pg_isready -U postgres -d seguimiento_y_medicion
```

Al terminar, detené el servicio o bajá los servicios de Compose:

```powershell
docker compose stop postgres
# Alternativamente:
docker compose down
```

`docker compose down` conserva el volumen de datos. No uses `docker compose down -v`
salvo que quieras borrar deliberadamente los datos locales de PostgreSQL.

## Run the tests

Desde la raíz del repositorio, iniciar PostgreSQL:

```powershell
docker compose up -d postgres
```

Aplicar las migraciones de proyectos, historias y esfuerzo; también deben estar instaladas las
migraciones de HU-04 que crean `sprints` y `sprint_stories`. La migración de métricas requiere
esas tablas:

```powershell
migrate -path db/migrations -database "$DATABASE_URL" up
```

Luego ejecutar las pruebas:

```powershell
go test ./...
```

El comando debe finalizar correctamente y ejecutar las pruebas unitarias y escenarios BDD.
La prueba de integración de `PostgreSQLMetricsStore` se omite si PostgreSQL no está disponible
o si faltan las tablas de HU-04 (`sprints` y `sprint_stories`) o la tabla de métricas.

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
   - cuando está aplicado el esquema de HU-04 y métricas, la prueba de integración se ejecuta
     contra PostgreSQL real.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio.
Las pruebas deben ejecutar las reglas reales de cálculo y persistencia, no reemplazarlas con
mocks o métodos exclusivos de test.
