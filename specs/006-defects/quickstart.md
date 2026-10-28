# Quickstart: Registro y trazabilidad de defectos

## Requisitos previos

- Go 1.27.1 instalado; verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- Herramienta `golang-migrate/migrate` instalada según el [README.md](../../README.md).
- HU-02, HU-04, FIX-HU-04, HU-20a y HU-20b (completa) integradas en `main`.
- PostgreSQL configurado con `docker-compose.override.yml` local. El compose base publica el puerto `5433` del
  host (`5432` dentro del contenedor); el override puede cambiarlo. Verificar el puerto efectivo con
  `docker compose port postgres 5432`.
- Configurar `DATABASE_URL` en la sesión de PowerShell con los valores locales, sin guardar ni compartir la
  contraseña:

  ```powershell
  $env:DATABASE_URL = "postgres://USUARIO:CLAVE@localhost:PUERTO/seguimiento_y_medicion?sslmode=disable"
  ```

No escribas claves personales en el repositorio ni compartas la salida completa de `docker compose config`.

## Preparar el entorno

Levantar PostgreSQL, revisar que esté saludable y confirmar el puerto publicado:

```powershell
docker compose up -d postgres
docker compose ps
docker compose port postgres 5432
```

Aplicar las migraciones pendientes:

```powershell
migrate -path db/migrations -database $env:DATABASE_URL up
```

No ejecutes `down 1` sobre una base compartida o con datos que quieras conservar. La reversión de la migración se
verifica en una base local desechable siguiendo T005; para el uso normal basta con aplicar `up`.

## Probar la pantalla manualmente

Se necesita un proyecto elegido como contexto activo, un Sprint con estado `Activo` y al menos una historia de ese
mismo proyecto. La historia puede estar en cualquier estado, pertenecer a un Sprint finalizado o no tener Sprint.
Iniciar la aplicación desde la misma sesión de PowerShell donde configuraste `DATABASE_URL`:

```powershell
go run ./src/cmd/server
```

1. Crear o seleccionar un proyecto y abrir `http://localhost:8080/contexto/proyecto` para elegirlo (HU-20b).
2. Crear una historia y un Sprint activo para ese proyecto. Si las pantallas de gestión todavía no están
   integradas, preparar datos locales mediante SQL respetando el esquema de [data-model.md](./data-model.md); no
   reutilizar datos de otro proyecto.
3. Abrir `http://localhost:8080/defectos`. Confirmar que el formulario muestra los campos descripción, severidad
   e historia relacionada, y que las opciones corresponden al proyecto activo.
4. Registrar una descripción breve, una severidad admitida y la historia creada. Confirmar la redirección a
   `/defectos`, el mensaje de éxito con el Sprint de detección y la fila nueva con estado `Abierto`. Recargar:
   no debe duplicar el defecto ni repetir el mensaje.
5. Probar un envío con campos vacíos y verificar que muestra los errores junto a cada campo, conserva los valores
   y no agrega filas. Probar una descripción de más de 1000 caracteres y una severidad no válida.
6. Cambiar a una historia `Hecho` de un Sprint finalizado (o sin Sprint), si está disponible. Debe poder elegirse
   y asociarse al defecto sin cambiar su estado ni asignación.
7. En un proyecto sin Sprint activo, abrir `Defectos`: debe aparecer el mensaje correspondiente y no el
   formulario. La lista visible debe contener solo defectos del proyecto seleccionado, con los más recientes
   primero.

Las pruebas automáticas no dependen de HU-21, HU-22 ni HU-23: crean sus propios proyectos, historias y Sprints.

## Ejecutar las pruebas

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al compartir una base.

## Escenarios de validación requeridos

Las pruebas deben cubrir como mínimo:

1. Migración: `up` y `down` en ambos sentidos; los `CHECK` de severidad, estado, descripción y coherencia entre
   estado y Sprint de resolución rechazan datos inválidos.
2. Servicio (unitarias y PostgreSQL real):
   - validación completa: descripción vacía, solo espacios, de 1000 y de 1001 caracteres; severidad vacía y fuera
     de la lista; historia vacía; todas las violaciones juntas;
   - registro válido: estado "Abierto", Sprint de detección = Sprint activo, sin Sprint de resolución, historia sin
     cambios;
   - historia "Hecho" de un Sprint finalizado y historia sin Sprint: aceptadas;
   - sin Sprint activo (sin Sprints o solo finalizados): `ErrNoActiveSprint` y nada guardado;
   - historia de otro proyecto o inexistente: `ErrStoryNotFound` y nada guardado;
   - un Sprint activo en el proyecto A no afecta al B;
   - falla a mitad del registro: se revierte todo;
   - registro y cierre de Sprint (HU-15) simultáneos: el defecto queda en el Sprint antes del cierre o se rechaza
     con `ErrNoActiveSprint`, nunca incoherente.
3. Catálogo (PostgreSQL real): lista del proyecto correcto, más recientes primero, con historia y Sprints; sin datos
   de otro proyecto; opciones de historia; número del Sprint activo.
4. Handler (con `cookiejar` y proyecto activo elegido por el flujo de HU-20b):
   - registro válido: 303 a `/defectos`, mensaje con el Sprint una sola vez, defecto "Abierto"; un campo de Sprint o
     de proyecto enviado se ignora; con dos proyectos solo registra en el activo;
   - cada una de las cuatro severidades;
   - varios errores a la vez, descripción inválida, historia ajena, inexistente o no numérica, severidad inválida:
     mensajes junto a cada campo, datos conservados, 422 y nada guardado;
   - sin Sprint activo: `GET` con el mensaje y sin formulario, `POST` con 409 y nada guardado;
   - falla de guardado: "La operación no pudo completarse", datos conservados y sin éxito;
   - lista con columnas, orden y "—" en el Sprint de resolución; lista vacía con el mensaje y el formulario;
   - descripción y título con `<script>`: se muestran como texto;
   - recarga después de registrar: no se crea un segundo defecto ni se repite el mensaje;
   - sin proyecto activo: redirección de HU-20b y nada guardado;
   - falla de lectura: página de error de la aplicación.
5. Seguridad: ninguna respuesta de error contiene texto de consultas ni rutas.

## Evidencia de aceptación

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las pruebas ejecutan las
reglas reales y el almacenamiento real; solo se usa un almacenamiento que falla para simular la falla de guardado y
de lectura.

Guardar la evidencia en `specs/006-defects/`. En PowerShell, ejecutar `chcp 65001` antes para que las tildes y la ñ
no salgan como `├®`:

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/006-defects/evidencia-red.txt
```

Esa primera ejecución se guarda antes de implementar, con las pruebas nuevas fallando. Al cerrar la historia, la
ejecución final y la cobertura:

```powershell
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/006-defects/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/006-defects/cobertura.txt
```
