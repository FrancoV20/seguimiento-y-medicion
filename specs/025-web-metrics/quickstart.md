# Quickstart: Interfaz web de métricas del Sprint

## Prerequisites

- Go 1.27.1 instalado; verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- Herramienta `golang-migrate/migrate` instalada según el [README.md](../../README.md).
- HU-07, HU-15, HU-17, HU-20a y HU-20b (completa) integradas en `main`; en tu rama, traerlas con
  `git merge origin/main`.
- Variable `DATABASE_URL` configurada. Ajustar el puerto al que publica tu `docker-compose.yml` o
  `docker-compose.override.yml` (por ejemplo 5432 o 5433):
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- No subir al repositorio claves personales; usar `docker-compose.override.yml`.

## Run the application

Esta feature no agrega migraciones; basta con las ya aplicadas:

```powershell
docker compose up -d postgres
migrate -path db/migrations -database $env:DATABASE_URL up
```

Para probar a mano hace falta un proyecto con al menos un Sprint cerrado:

1. Crear el proyecto (HU-21) y elegirlo en `http://localhost:8080/contexto/proyecto` (HU-20b).
2. Crear historias (HU-22) con horas estimadas (HU-17), armar un Sprint (HU-23), pasarlas a "Hecho" (HU-14),
   cargar horas (HU-24) y cerrar el Sprint en "Cierre de Sprint" (HU-15). Si esas pantallas todavía no están
   integradas, insertar los datos por SQL respetando el esquema de `data-model.md`.
3. Abrir `http://localhost:8080/metricas`.

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable"
go run ./src/cmd/server
```

Las pruebas automáticas no dependen de las demás pantallas: crean sus propios proyectos, historias, Sprints, horas
y cierres.

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al compartir una base.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Formato (unitarias): desviación positiva, negativa y cero con dos decimales, coma y signo; "No disponible";
   velocidad cero; alerta de cálculo parcial con X de Y; elección del Sprint (válido, inválido, activo, ajeno, sin
   lista).
2. Adaptador de HU-07 (servicio real y PostgreSQL real): velocidad de las historias "Hecho"; desviación con valores
   conocidos (por ejemplo 10 horas estimadas y 12 reales dan `+20,00 %`); sin horas estimadas; cálculo parcial con
   X de Y; Sprint cerrado por HU-15 sin historias completadas (velocidad cero); las historias devueltas no cuentan.
3. Catálogo (PostgreSQL real): solo Sprints finalizados del proyecto, más reciente primero; el Sprint activo no
   figura y su número se informa; otro proyecto no se mezcla; proyecto sin Sprints.
4. Handler (con `cookiejar` y proyecto activo elegido por el flujo de HU-20b):
   - Sprint finalizado: velocidad y desviación correctas con datos reales; por defecto el más reciente;
   - sin horas estimadas: "No disponible" y velocidad visible;
   - cálculo parcial: alerta con X de Y;
   - velocidad cero: `0 Story Points` y "No disponible";
   - lista de Sprints y métricas del Sprint elegido con `?sprint=ID`; `sprint` inválido, activo o de otro proyecto se
     ignora;
   - sin Sprints finalizados, con y sin Sprint activo;
   - Sprint Goal con `<script>`: se ve como texto;
   - falla de lectura o de cálculo: página de error de la aplicación;
   - sin proyecto activo: redirección de HU-20b;
   - `POST` a `/metricas`: página 405.
5. Seguridad: ninguna respuesta de error contiene texto de consultas ni rutas.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las pruebas ejecutan las
reglas reales y el almacenamiento real; solo se usa una fuente que falla para simular las fallas de lectura y de
cálculo.

Guardar la evidencia en `specs/025-web-metrics/`. En PowerShell, ejecutar `chcp 65001` antes para que las tildes y
la ñ no salgan como `├®`:

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/025-web-metrics/evidencia-red.txt
```

Esa primera ejecución se guarda antes de implementar, con las pruebas nuevas fallando. Al cerrar la historia, la
ejecución final y la cobertura:

```powershell
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/025-web-metrics/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/025-web-metrics/cobertura.txt
```