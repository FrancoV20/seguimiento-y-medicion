# Quickstart: Formularios seguros y contexto de trabajo

## Prerequisites

- HU-20a integrada: servidor, layout, `Section`, `Server.Register`, `Server.Use`, `ActiveContext`,
  `Fail` y `RenderError` disponibles.
- Go 1.27.1 y Docker Desktop instalados y en ejecución.
- Herramienta `golang-migrate/migrate` instalada según el [README.md](../../README.md).
- Migración `001` (proyectos e integrantes, HU-01) aplicada. Esta feature no agrega migraciones.
- Variable `DATABASE_URL` configurada. Ajustar el puerto al que publica tu `docker-compose.yml`
  o `docker-compose.override.yml` (por ejemplo 5432 o 5433):
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.

## Run the tests

Desde la raíz del repositorio:

```powershell
docker compose up -d postgres
migrate -path db/migrations -database $env:DATABASE_URL up
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al
compartir una base. Si la base local ya aplicó una migración de número mayor y falta una de
número menor, recrearla con `docker compose down -v` y volver a migrar.

## Datos de prueba

Las pruebas de integración y BDD crean sus propios proyectos e integrantes (fixtures) y los
eliminan al terminar. Para probar a mano antes de que exista HU-21, crear dos proyectos con varios
integrantes usando el caso de uso `CreateProject` de HU-01 desde una prueba o con `INSERT`
ajustados al esquema real de `001_*.up.sql` (tablas `projects`, `members` y `project_members`).

Mientras no existan las secciones reales (Backlog, Sprints, Esfuerzo, Métricas), las pruebas
registran secciones de prueba: una sin requisitos, una con `RequiresProject` y una con
`RequiresMember`.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Mensajes y formularios:
   - formato por tipo de mensaje, con etiqueta de texto además del color;
   - errores junto al campo con datos conservados;
   - texto único "La operación no pudo completarse", sin mensaje de éxito;
   - envío válido: 303 y mensaje una sola vez; recarga sin duplicar.
2. Seguridad:
   - texto con `<script>` mostrado como texto;
   - envío con `Sec-Fetch-Site: cross-site` o `Origin` ajeno rechazado con 403 y mensaje claro;
     mismo origen aceptado;
   - (extra, opcional) cuerpo mayor a 1 MiB rechazado con 413 y cabeceras de seguridad.
3. Selección de proyecto:
   - elegir un proyecto y ver solo sus datos con dos proyectos de datos distintos;
   - mantener la selección al navegar entre secciones y ver el nombre en la estructura común;
   - sin proyectos: "Aún no hay proyectos" y acceso a crear el primero solo si la sección
     Proyectos está registrada;
   - con un solo proyecto: no se autoselecciona;
   - cambiar de proyecto limpia el integrante; volver a elegir el mismo lo conserva;
   - cerrar el navegador (cookie de sesión nueva) vuelve a pedir el proyecto.
4. Datos inválidos:
   - identificador inexistente, no numérico, negativo o muy grande: mensaje claro, selección
     anterior conservada, sin error interno;
   - cookie manipulada: tratada como ausencia y borrada;
   - proyecto eliminado después de seleccionado: selección limpiada con advertencia.
5. Identificación del integrante:
   - el selector muestra solo integrantes del proyecto activo;
   - la identificación se mantiene y se muestra en el layout;
   - acción que requiere integrante sin identificarse: se pide identificarse y se vuelve;
   - integrante de otro proyecto: acción rechazada y identificación limpiada;
   - integrante quitado del proyecto: identificación limpiada con advertencia.
6. Secciones protegidas:
   - sin proyecto, las secciones con `RequiresProject` redirigen a la selección con mensaje;
   - al elegir, vuelve a la sección pedida;
   - destinos `volver_a` externos o malformados (`//otro.com`, `https://...`, `\\`) van a `/`;
   - secciones sin requisitos accesibles sin contexto.
7. Disponibilidad:
   - con la base de datos caída no se limpia la selección y se muestra el 503 de HU-20a.
8. PostgreSQL:
   - `PostgreSQLContextStore` lista proyectos e integrantes y resuelve el contexto contra
     PostgreSQL real;
   - `ResolveContext` distingue "no existe" de "no disponible".

## Manual check

1. Abrir `http://localhost:8080`, ir a una sección protegida y comprobar la redirección a la
   selección de proyecto.
2. Elegir proyecto e integrante y navegar entre secciones.
3. Enviar un formulario de prueba válido, recargar y comprobar que no se duplica y que el mensaje
   de éxito no reaparece.
4. Cerrar todas las ventanas del navegador, volver a abrir y comprobar que se pide elegir de nuevo.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las
pruebas deben ejecutar el servidor, el middleware y el almacenamiento reales, no reemplazarlos con
mocks o métodos exclusivos de test.
