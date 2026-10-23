# Quickstart: Interfaz web de registro de esfuerzo

## Prerequisites

- Go 1.27.1 instalado; verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- Herramienta `golang-migrate/migrate` instalada según el [README.md](../../README.md).
- HU-03, HU-04, FIX-HU-04, HU-20a y HU-20b (completa) integradas en `main`; en tu rama, traerlas con
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

Para probar a mano hace falta un proyecto con integrantes, un Sprint activo con historias y una identificación:

1. Cargar los integrantes con `db/seeds/members.sql` si no están (HU-21) y crear el proyecto (HU-21).
2. Elegir el proyecto en `http://localhost:8080/contexto/proyecto` e identificarse como integrante (HU-20b).
3. Crear las historias (HU-22) y un Sprint con ellas (HU-23). Si esas pantallas todavía no están integradas, insertar
   los datos por SQL respetando el esquema de `data-model.md`.
4. Abrir `http://localhost:8080/esfuerzo`.

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable"
go run ./src/cmd/server
```

Las pruebas automáticas no dependen de HU-21, HU-22 ni HU-23: crean sus propios proyectos, integrantes, historias,
Sprints y registros.

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al compartir una base.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Formulario (unitarias): fecha vacía, con formato inválido y válida; historia vacía y no numérica; horas con coma,
   con punto, negativas, cero, mayores de 24, vacías, con letras y en formato científico; actividad con espacios;
   un error por campo con los de HU-03 combinados con los propios.
2. Adaptador de HU-03 (servicio real): todas las violaciones juntas con sus campos y mensajes exactos; registro
   válido; integrante faltante.
3. Catálogo (PostgreSQL real):
   - solo historias del Sprint activo del proyecto (ni "Pendiente", ni de un Sprint finalizado, ni de otro
     proyecto), con horas acumuladas y cero cuando no hay registros;
   - historial con integrante, fecha, actividad y horas, más recientes primero, y total igual a la suma;
   - historia ajena al Sprint activo: "no encontrada"; proyecto sin Sprint activo.
4. Handler (con `cookiejar`, proyecto activo e integrante identificado por el flujo de HU-20b):
   - carga válida: 303 a `/esfuerzo?historia=ID`, mensaje una sola vez, registro a nombre del integrante
     identificado y acumulado actualizado; un campo de integrante enviado se ignora;
   - fecha de hoy por defecto con un reloj fijo;
   - varios errores a la vez: todos los mensajes de HU-03 junto a su campo, datos conservados, 422 y nada registrado;
   - historia no seleccionada, de otro proyecto, "Pendiente", de un Sprint finalizado, inexistente o no numérica:
     error junto a `historia` y nada registrado;
   - sin Sprint activo: mensaje y sin formulario; `POST` con 409 y nada registrado;
   - sin integrante identificado: redirección de HU-20b y regreso a `/esfuerzo`; nada registrado;
   - falla de guardado: "La operación no pudo completarse", datos conservados y sin éxito;
   - el Sprint se cierra entre abrir y enviar: HU-03 rechaza y el error aparece junto a la historia;
   - historial con registros de dos integrantes y total; historia sin registros con el mensaje y total en cero;
     `historia` inválida en la dirección: se ignora;
   - actividad con `<script>`: se ve como texto;
   - recarga después de guardar: no se reenvía ni se repite el mensaje;
   - sin proyecto activo: redirección de HU-20b y nada registrado;
   - falla de lectura: página de error de la aplicación.
5. Seguridad: ninguna respuesta de error contiene texto de consultas ni rutas.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las pruebas ejecutan las
reglas reales y el almacenamiento real; solo se usa un almacenamiento que falla para simular la falla de guardado y
de lectura.

Guardar la evidencia en `specs/024-web-effort/`. En PowerShell, ejecutar `chcp 65001` antes para que las tildes y la
ñ no salgan como `├®`:

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/024-web-effort/evidencia-red.txt
```

Esa primera ejecución se guarda antes de implementar, con las pruebas nuevas fallando. Al cerrar la historia, la
ejecución final y la cobertura:

```powershell
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/024-web-effort/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/024-web-effort/cobertura.txt
```