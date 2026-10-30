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

## Estado del proyecto

Elegí primero el proyecto desde **Cambiar proyecto** y abrí `http://localhost:8080/estado`, o la sección
**Estado del proyecto** del menú. La pantalla es de solo lectura: calcula todo al abrirla, con los datos y la fecha de
ese momento, y no guarda nada. Muestra cuatro cosas del proyecto activo:

1. **Nombre y fechas**: el proyecto y sus fechas de inicio y fin, en formato dd/mm/aaaa.
2. **Estado**: **No iniciado**, **En curso** o **Finalizado**, según la fecha de hoy.
3. **Avance**: **X de Y historias completadas (P %)**, donde X son las historias en Hecho, Y el total de historias del
   proyecto y P el porcentaje con dos decimales y coma (por ejemplo, 4 de 8 es 50,00 %).
4. **Historias por estado**: una tabla con la cantidad de historias en Pendiente, En Sprint, En progreso y Hecho. Un
   estado sin historias muestra 0.
5. **Sprint en curso**: el número del Sprint activo, su objetivo y sus fechas.

### Cómo se calcula el estado

| Estado | Cuándo se muestra |
|---|---|
| No iniciado | Hoy es anterior a la fecha de inicio. |
| En curso | Hoy está entre la fecha de inicio y la de fin; los dos días cuentan. |
| Finalizado | Hoy es posterior a la fecha de fin. |

Se comparan los días, no las horas: el mismo día de inicio o de fin el proyecto ya figura **En curso**. Recargar la
pantalla vuelve a calcular, así que un proyecto puede pasar de En curso a Finalizado al cambiar el día. El estado no
depende de las historias ni de los Sprints.

### Mensajes

| Mensaje | Qué significa |
|---|---|
| `0 de 0 historias completadas (0,00 %)` | El proyecto todavía no tiene historias. |
| `No hay un Sprint en curso` | El proyecto no tiene un Sprint activo; el resto de la información se muestra igual. |
| `Sin proyecto seleccionado` | Primero elegí un proyecto: la aplicación te lo pide y, al elegirlo, vuelve a esta pantalla. |
| `El servicio no está disponible temporalmente` | La base de datos no responde. Intentá de nuevo en unos minutos. |
| `Ocurrió un error inesperado. Intentá nuevamente en unos minutos.` | Hubo un problema al leer los datos; no se muestra información parcial. |
| `La dirección que buscás no existe o ya no está disponible.` | El proyecto elegido ya no existe. Elegí otro. |