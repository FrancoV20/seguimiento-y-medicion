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

### Modificar un proyecto

En la lista de **Gestión de Proyectos**, presioná **Modificar** en la fila del proyecto. El
formulario se abre con el nombre, las fechas y los integrantes actuales.

1. Cambiá el nombre o las fechas, o marcá y desmarcá integrantes. Solo aparecen los integrantes que
   ya existen en el sistema.
2. Presioná **Guardar**. Para salir sin cambios, presioná **Cancelar**.

Si se guarda, aparece **Proyecto modificado exitosamente** y volvés a la lista. Si el proyecto es tu
contexto de trabajo, el nombre nuevo se ve arriba sin volver a elegirlo. Se guarda todo junto: si
algo falla, no cambia nada.

Además de los mensajes de la creación, al modificar pueden aparecer:

| Mensaje | Qué hacer |
|---|---|
| `No se puede quitar a [nombre]: tiene horas registradas en este proyecto` | Ese integrante ya cargó horas en historias del proyecto y no se puede quitar. Volvé a marcarlo. |
| `El Sprint N queda fuera de las nuevas fechas del proyecto` | Las fechas nuevas dejan fuera a ese Sprint. Elegí fechas que lo incluyan. |

Si quitás a un integrante que estaba identificado, la próxima vez que navegue se le pide elegir
otro integrante.

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
| `La descripción contiene caracteres no válidos` | Quitá los caracteres invisibles o dañados (suelen aparecer al pegar texto copiado de otro programa) y escribí la descripción como texto común. |
| `Seleccione la severidad` | Elegí una de las severidades disponibles. |
| `La severidad no es válida` | Seleccioná una severidad de la lista. |
| `Seleccione la historia relacionada` | Elegí una historia del proyecto activo. |
| `La historia no pertenece al proyecto` | Cambiá a una historia del proyecto activo. |
| `No hay un Sprint activo; no se puede registrar un defecto` | Iniciá un Sprint para el proyecto antes de registrar. |
| `La operación no pudo completarse` | Hubo un problema al guardar. Intentá nuevamente más tarde. |

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

## Estimaciones

Elegí el proyecto desde **Cambiar proyecto** y abrí `http://localhost:8080/estimaciones`. La pantalla
lista las historias del proyecto, las más recientes primero, con el título, el estado, los Story Points,
las **horas estimadas** y las **horas reales** registradas en el Registro de esfuerzo. Lo que todavía no
se estimó aparece como **Sin estimar** (nunca como 0); las horas reales aparecen como 0 si nadie cargó
horas.

### Registrar o modificar las horas estimadas

Escribí las horas en el campo de la fila y presioná **Guardar**. Se acepta coma o punto como separador
decimal (`1,5` o `1.5`), con hasta dos decimales y hasta 9999,99 horas, sin separadores de miles. La
primera vez se muestra **Horas estimadas registradas** y las siguientes **Horas estimadas actualizadas**.
Las horas se muestran siempre con coma. Cambiar la estimación no modifica las horas reales ni ningún otro
dato de la historia.

Las historias de un Sprint finalizado (las **Hecho** al cerrar el Sprint) muestran el campo deshabilitado
con el motivo **Sprint finalizado**: su estimación ya no se puede cambiar. Las historias que el cierre
devolvió al Backlog se pueden volver a estimar.

### Corregir errores

- **El valor de horas no es válido**: el valor es 0, negativo, vacío, no numérico o tiene más de dos
  decimales.
- **Las horas estimadas no pueden superar 9999,99**.
- **La historia no existe**: la historia no pertenece al proyecto elegido.
- **La operación no pudo completarse**: no se guardó nada; volvé a intentarlo más tarde.

Ante un error, lo que escribiste queda en el campo para corregirlo y el valor anterior no cambia. Si el
proyecto no tiene historias, la pantalla muestra **Aún no hay historias**.

## Cierre de Sprint

Elegí el proyecto desde **Cambiar proyecto** y abrí `http://localhost:8080/cierre`. La pantalla muestra
el Sprint activo, las historias **Hecho** que permanecerán en el Sprint y las historias **En Sprint** o
**En progreso** que volverán al Backlog en estado **Pendiente**. También muestra los Story Points
completados sobre los planificados.

Antes de confirmar, revisá el resumen: el cierre no se puede deshacer. Al presionar **Cerrar Sprint**,
las historias incompletas pierden su asignación y quedan disponibles para un Sprint posterior; las
historias **Hecho** conservan su estado y su asignación al Sprint finalizado. El sistema guarda además
una foto de los estados y Story Points para las métricas históricas.

Si no hay un Sprint activo, la pantalla informa **No hay un Sprint activo para cerrar**. Si HU-07 detecta
datos inválidos en historias completadas, corregí las historias indicadas antes de reintentar. Un segundo
envío de un Sprint ya cerrado se rechaza; si falla el guardado, no se aplican cambios parciales y se
muestra **La operación no pudo completarse**.