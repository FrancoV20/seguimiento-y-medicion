# Manual breve de usuario

## Gestión de Proyectos

Abrí `http://localhost:8080/proyectos` para ver los proyectos existentes y crear uno nuevo. La
pantalla muestra primero los proyectos más recientes, con su nombre, fechas y cantidad de
integrantes.

![Pantalla Gestión de Proyectos](./gestion-proyectos.png)

### Crear un proyecto

1. Escribí el nombre del proyecto.
2. Marcá uno o más integrantes disponibles.
3. Elegí las fechas de inicio y fin.
4. Presioná **Crear**. La fecha de fin debe ser posterior a la fecha de inicio.

Si la creación se completa, aparece **Proyecto creado exitosamente** y el proyecto nuevo queda al
principio de la lista. La creación no selecciona el proyecto activo.

### Corregir errores

Los mensajes de validación aparecen junto al campo que hay que corregir. El formulario conserva los
valores ingresados:

| Mensaje | Qué hacer |
|---|---|
| `project name cannot be empty` | Escribí un nombre; no puede quedar vacío ni contener solo espacios. |
| `El nombre no puede superar los 255 caracteres` | Acortá el nombre. |
| `the member list is required` | Marcá al menos un integrante. |
| `Falta la fecha de inicio` | Ingresá la fecha de inicio. |
| `Falta la fecha de fin` | Ingresá la fecha de fin. |
| `La fecha no tiene un formato válido` | Volvé a elegir la fecha con el control del navegador. |
| `El integrante seleccionado no es válido` | Elegí integrantes que aparezcan en la lista. |
| `La fecha de finalización debe ser posterior a la fecha de inicio` | Corregí la fecha de fin: debe ser posterior a la de inicio. |
| `La operación no pudo completarse` | Hubo un problema al guardar. Intentá nuevamente más tarde. |

Si todavía no hay integrantes cargados, la pantalla informa **No hay integrantes disponibles** y
no permite enviar el formulario.

## Registro y trazabilidad de defectos

Elegí primero el proyecto desde **Cambiar proyecto** y abrí `http://localhost:8080/defectos`. Para
registrar un defecto, el proyecto debe tener un Sprint activo y una historia relacionada. Completá:

1. **Descripción**: texto de 1 a 1000 caracteres; se eliminan los espacios al principio y al final.
2. **Severidad**: Crítica, Alta, Media o Baja.
3. **Historia relacionada**: cualquier historia del proyecto activo, incluso si está Hecho, pertenece
   a un Sprint finalizado o no tiene Sprint.

Al registrar, el defecto queda **Abierto** y se asocia al Sprint activo como Sprint de detección. La
historia conserva su estado y asignación. La confirmación indica el Sprint utilizado; recargar la
página no vuelve a registrar el defecto ni repite el mensaje.

La lista muestra identificador, descripción, severidad, estado, historia relacionada y Sprints de
detección y resolución. Los defectos más recientes aparecen primero; `—` indica que todavía no hay
Sprint de resolución. Solo se muestran defectos del proyecto activo. Si no hay defectos, la pantalla
muestra **Aún no hay defectos** y permite registrar el primero.

### Errores de registro

Los errores aparecen junto al campo correspondiente y el formulario conserva los valores ingresados.
Si se detectan varios problemas, se muestran juntos y no se guarda el defecto.

| Mensaje | Qué hacer |
|---|---|
| `La descripción es obligatoria` | Escribí una descripción con contenido. |
| `La descripción no puede superar 1000 caracteres` | Acortá la descripción a 1000 caracteres o menos. |
| `Seleccione la severidad` | Elegí una de las severidades disponibles. |
| `La severidad no es válida` | Seleccioná una severidad de la lista. |
| `Seleccione la historia relacionada` | Elegí una historia del proyecto activo. |
| `La historia no pertenece al proyecto` | Cambiá a una historia del proyecto activo. |
| `No hay un Sprint activo; no se puede registrar un defecto` | Iniciá un Sprint para el proyecto antes de registrar. |
| `La operación no pudo completarse` | Hubo un problema al guardar. Intentá nuevamente más tarde. |
