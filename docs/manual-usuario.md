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

## Registro de esfuerzo

Abrí `http://localhost:8080/esfuerzo` para cargar las horas que trabajaste en una historia del
Sprint activo. Antes elegí el proyecto (**Elegir proyecto**) e identificate (**Identificarme**): si
no lo hiciste, la aplicación te lo pide y después vuelve a esta pantalla. Las horas quedan siempre a
tu nombre.

### Cargar horas

1. Elegí la historia. Solo aparecen las del Sprint activo del proyecto.
2. Revisá la fecha: por defecto es la de hoy y no puede ser posterior.
3. Describí la actividad.
4. Escribí las horas, con coma o punto y hasta dos decimales (más de 0 y hasta 24).
5. Presioná **Registrar horas**.

Si la carga se completa, aparece **Esfuerzo registrado correctamente** y se muestra el historial de
la historia con el total actualizado.

### Ver el historial

Tocá el nombre de una historia en la tabla para ver quién cargó horas, en qué fecha, en qué
actividad y el total. Las cargas más recientes aparecen primero. La tabla de historias muestra las
horas acumuladas de cada una.

### Corregir errores

Todos los errores aparecen juntos, cada uno junto a su campo, y el formulario conserva lo escrito:

| Mensaje | Qué hacer |
|---|---|
| `Seleccione una historia` | Elegí una historia de la lista. |
| `La historia no pertenece al Sprint activo` | Elegí una historia del Sprint activo del proyecto. |
| `La historia no está asignada a un Sprint activo` | El Sprint se cerró mientras cargabas: recargá la pantalla. |
| `La fecha es requerida` | Ingresá la fecha. |
| `La fecha no es válida` | Volvé a elegir la fecha con el control del navegador. |
| `La fecha no puede ser posterior a la fecha actual` | Elegí hoy o una fecha anterior. |
| `La descripción de la actividad es obligatoria` | Describí la actividad. |
| `El valor de horas no es válido` | Escribí un número mayor que 0 y hasta 24, con hasta dos decimales. |
| `La operación no pudo completarse` | Hubo un problema al guardar. Intentá nuevamente más tarde. |

Si el proyecto no tiene un Sprint activo, la pantalla muestra **No hay un Sprint activo** y no
permite cargar horas. La pantalla no permite editar ni borrar cargas.
