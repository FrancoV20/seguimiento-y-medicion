# Quickstart: Interfaz web del Product Backlog

## Prerequisites

- Go 1.27.1 instalado; verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- Herramienta `golang-migrate/migrate` instalada según el [README.md](../../README.md).
- HU-02, HU-20a y HU-20b (completa) integradas en `main`; en tu rama, traerlas con
  `git merge origin/main`.
- Variable `DATABASE_URL` configurada. Ajustar el puerto al que publica tu `docker-compose.yml` o
  `docker-compose.override.yml` (por ejemplo 5432 o 5433):
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- No subir al repositorio claves personales; usar `docker-compose.override.yml`.

## Run the application

Desde la raíz del repositorio, iniciar PostgreSQL y aplicar migraciones:

```powershell
docker compose up -d postgres
migrate -path db/migrations -database $env:DATABASE_URL up
```

Para probar a mano hace falta un proyecto activo:

1. Si no hay integrantes, cargarlos con la semilla de HU-21 (`db/seeds/members.sql`).
2. Crear un proyecto en `http://localhost:8080/proyectos` (HU-21).
3. Elegirlo como proyecto activo en `http://localhost:8080/contexto/proyecto` (HU-20b).

Iniciar el servidor y abrir `http://localhost:8080/backlog`:

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable"
go run ./src/cmd/server
```

Si HU-21 todavía no está integrada, las pruebas automáticas no la necesitan: crean sus propios
proyectos e historias. Esta feature no agrega migraciones; si alguna vez hiciera falta una, toma el
último número de `main` más uno (hoy la próxima es la 008), sin importar el número de la historia.

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al
compartir una base.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Alta válida:
   - historia con título, descripción, prioridad y criterios: mensaje de éxito y aparece en el
     Backlog con estado "Pendiente", en el proyecto activo;
   - con dos proyectos, la historia queda solo en el activo;
   - recargar la página después de guardar: no se duplica ni se repite el mensaje.
2. Criterios dinámicos:
   - "Agregar criterio" muestra un campo más, conserva lo escrito y no guarda;
   - "Quitar" elimina ese campo y no permite quitar el último;
   - los criterios en blanco se ignoran al guardar;
   - Enter dentro de un campo guarda y no agrega ni quita criterios.
3. Story Points:
   - sin Story Points: la historia se crea y se muestra la advertencia; en la lista figura "Sin
     estimar";
   - Story Points inválidos: mensaje junto al campo y no se crea;
   - horas estimadas válidas: la historia se crea con ellas y se ven en la lista;
   - horas estimadas inválidas (cero, negativas, no numéricas y más de dos decimales): mensaje junto al
     campo, datos conservados, sin crear y sin redondeo silencioso;
   - prioridad y Story Points solo aceptan los valores de la migración `002`.
4. Errores de validación:
   - sin criterios con texto: error junto al campo, datos conservados, sin crear;
   - título o descripción vacíos y prioridad inválida: mensajes de HU-02 junto a cada campo.
5. Falla de guardado:
   - "La operación no pudo completarse", datos conservados, sin mensaje de éxito y sin la historia en
     el Backlog.
6. Contexto:
   - sin proyecto activo, guardar redirige a la selección de proyecto con el mensaje de HU-20b y no
     guarda nada;
   - un identificador de proyecto enviado en el formulario se ignora.
7. Lista:
   - título, prioridad, Story Points y horas estimadas (cada uno o "Sin estimar") y estado, con las
     más recientes primero y solo
     del proyecto activo;
   - sin historias: "Aún no hay historias" y el formulario;
   - falla de lectura: página de error de la aplicación, no un formulario incompleto.
8. Seguridad y detalles internos:
   - un título con `<script>` se ve como texto;
   - ninguna respuesta de error contiene texto de consultas ni rutas.
9. PostgreSQL:
   - `PostgreSQLStoryCatalog` lista las historias de un proyecto contra PostgreSQL real, con
     fixtures propias, sin mezclar proyectos.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las
pruebas deben ejecutar las reglas reales de HU-02 y el almacenamiento real, no reemplazarlos con
mocks; solo se usa un almacenamiento que falla para simular la falla de guardado y de lectura.

Guardar la evidencia en `specs/022-web-backlog/`. En PowerShell, ejecutar `chcp 65001` antes para
que las tildes y la ñ no salgan como `├®`:

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/022-web-backlog/evidencia-red.txt
```

Esa primera ejecución se guarda antes de implementar, con las pruebas nuevas fallando. Al cerrar la
historia, la ejecución final y la cobertura:

```powershell
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/022-web-backlog/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/022-web-backlog/cobertura.txt
```
