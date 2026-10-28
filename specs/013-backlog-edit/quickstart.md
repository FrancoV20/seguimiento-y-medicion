# Guía rápida: HU-13 – Consulta y Edición del Product Backlog (web)

## Requisitos previos

- Go 1.27.1 (`go version`), Docker Desktop en ejecución y `golang-migrate/migrate` según el
  [README.md](../../README.md).
- HU-02, HU-04, HU-20a, HU-20b y HU-22 en `main`, ya integradas.
- HU-15 (cierre de Sprint) y HU-17 (horas estimadas) también tienen que estar en `main`: HU-13 se implementa
  después de ellas (decisión del equipo, 27/10). Comprobarlo con `git log --oneline main` antes de crear la rama.
- Rama: `git switch main; git pull; git switch -c HU-13`.
- `DATABASE_URL` con el puerto de tu `docker-compose.yml` o `docker-compose.override.yml`:
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- Base al día: `migrate -path db/migrations -database $env:DATABASE_URL up`. HU-13 no agrega migraciones.

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run "HU13|UpdateStory|StoryStatuses|ValidateStoryFields|StoryForEdit|StoryCatalog|Backlog" -v ./src/backlog/... ./src/web/... ./test/...
go test -count=1 -p 1 -cover ./src/backlog/...
```

`-count=1` evita la caché y `-p 1` evita que los paquetes de integración se pisen en la misma base.

## Escenarios de validación requeridos

1. **Dominio** con `InMemoryBacklogStore`:
   - `StoryStatuses`;
   - `validateStoryFields`: orden de los errores, recorte y numeración de los criterios;
   - `UpdateStory`:
     - identificadores en 0;
     - cada error de HU-02;
     - errores del almacén que pasan sin cambios;
     - falla genérica que se traduce a `ErrPersistence`;
     - advertencia sin Story Points.
   - Las pruebas de `CreateStory` siguen en verde.
2. **Adaptador** (PostgreSQL):
   - editar los cinco datos sin tocar `status`, `estimated_hours`, `active_sprint_id` ni `created_at`;
   - reordenar, agregar y quitar criterios;
   - quitar los Story Points;
   - historia inexistente;
   - historia de otro proyecto;
   - Sprint finalizado;
   - Sprint activo con una historia "Hecho";
   - falla a mitad sin cambios;
   - guardado sin cambios;
   - dos guardados simultáneos;
   - espera al cierre de un Sprint en los dos órdenes;
   - cambio de asignación durante el guardado.
3. **Catálogo** (PostgreSQL):
   - orden por prioridad, fecha de creación e identificador;
   - `InFinishedSprint`;
   - `StoryForEdit`: con criterios en orden, sin criterios, inexistente y de otro proyecto.
4. **Handlers**:
   - los escenarios de la spec con sus códigos (200, 303, 422, 409, 404, 500);
   - "404 antes que 422";
   - filtro llevado a la edición y validado;
   - `horas_estimadas` ignorado;
   - texto con `<script>` escapado;
   - falla de lectura;
   - recarga sin reenviar;
   - rendimiento.
5. **HU-22**: las tres pruebas del orden, actualizadas al orden nuevo, y el resto de sus pruebas sin cambios.

## Verificación manual

1. Levantar la aplicación (`go run ./src/cmd/server`), elegir un proyecto (`/contexto/proyecto`) y abrir
   `http://localhost:8080/backlog`.
2. Crear «Login» (Media), «Pagos» (Alta), «Reportes» (Baja) y «Perfil» (Media), en ese orden. La lista muestra
   «Pagos», «Login», «Perfil», «Reportes».
3. Filtrar por "Pendiente": se ven las cuatro. Filtrar por "Hecho": "No hay historias con ese estado". Elegir
   "Todos": se ve la lista completa.
4. Abrir `http://localhost:8080/backlog?estado=Archivada`: "El estado no es válido" y la lista completa.
5. Filtrar por "Pendiente" y presionar "Editar" en «Login». El formulario muestra sus datos. Cambiar el título,
   agregar y quitar un criterio (se conserva lo escrito) y guardar. Se vuelve a la lista filtrada con "Historia
   modificada correctamente".
6. Editar «Perfil», elegir "Sin estimar" y guardar: aparece la advertencia de estimación pendiente.
7. Borrar el título y guardar: "el título es obligatorio" junto al título, con los datos conservados.
8. En SQL, asignar «Pagos» a un Sprint y marcarlo `Finalizado`:

   ```sql
   UPDATE sprints SET status = 'Finalizado' WHERE id = …;
   ```

   En la lista, "Editar" de «Pagos» aparece deshabilitado con "Sprint finalizado". Abrir su dirección de edición
   vuelve a la lista con la advertencia.
9. Abrir `/backlog/999999/editar` y `/backlog/abc/editar`: página 404.
10. Salir del proyecto activo y abrir la dirección de edición de «Login»: se pide elegir proyecto y, al elegirlo,
    se vuelve a la edición.

## Evidencias a guardar en el PR

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 -Append specs/013-backlog-edit/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/013-backlog-edit/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/013-backlog-edit/cobertura.txt
```

`evidencia-red.txt` se completa por secciones: Fundamentos, Listado y filtro, Edición, Validaciones y Reglas y
errores. Cada sección se guarda antes de escribir el código de su bloque.
