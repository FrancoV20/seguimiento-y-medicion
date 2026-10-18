# Feature Specification: Interfaz web de proyectos

**Feature Branch**: `HU-21`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "HU-21 / Interfaz Web de Proyectos (presenta HU-01): Como Usuario, quiero crear y ver proyectos desde pantallas web, para gestionarlos sin comandos. Depende de HU-01, HU-20a y de la parte de formularios de HU-20b."

> **Alcance**: HU-21 presenta en el navegador lo que HU-01 ya resuelve (modelo, validaciones, alta y
> persistencia). Muestra la lista de proyectos y el formulario de alta en una sola pantalla,
> "Gestión de Proyectos". No incluye modificar proyectos (HU-11), consultar su estado (HU-12),
> seleccionar el proyecto activo (HU-20b) ni dar de alta integrantes (ver Assumptions).
>
> **Pendientes de HU-01 que se cierran acá**: el acceso al formulario desde Gestión de Proyectos
> (FR-001 de HU-01) y los criterios de usabilidad SC-003 y SC-004 (tiempo para completar el alta y
> éxito al primer intento).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Crear un proyecto desde el formulario web (Priority: P1)

Como Usuario, quiero completar nombre, integrantes y fechas en un formulario y crear el proyecto,
para gestionarlo sin comandos.

**Why this priority**: Es el flujo principal de HU-01 hecho consumible: sin esta pantalla el
sistema no tiene proyectos con los que trabajar y HU-22 a HU-25 no tienen datos que mostrar.

**Independent Test**: Con integrantes cargados en el sistema, completar el formulario con datos
válidos y verificar que el proyecto aparece en la lista con el mensaje de éxito; repetir con fechas
iguales, recargando la página y sin integrantes disponibles.

**Acceptance Scenarios**:

1. **Given** que estoy en "Gestión de Proyectos" y existen integrantes disponibles, **When**
   completo nombre, integrantes y fechas y presiono "Crear", **Then** veo el mensaje de éxito y el
   proyecto aparece en la lista.
2. **Given** que ingreso la misma fecha de inicio y de fin, **When** presiono "Crear", **Then** el
   proyecto se crea, porque no hay una finalización anterior al inicio.
3. **Given** que acabo de crear un proyecto, **When** recargo la página, **Then** no se crea un
   segundo proyecto ni se repite el mensaje de éxito.
4. **Given** que no hay integrantes cargados en el sistema, **When** abro "Gestión de Proyectos",
   **Then** veo el aviso "No hay integrantes disponibles" y no puedo crear un proyecto.

---

### User Story 2 - Corregir errores sin perder los datos (Priority: P1)

Como Usuario, quiero ver cada error junto a su campo y conservar lo que escribí, para corregirlo
sin volver a cargar todo.

**Why this priority**: Un formulario que borra los datos al fallar hace que el alta sea tediosa y
propensa a abandonarse; las reglas de HU-01 deben verse tal como están definidas.

**Independent Test**: Enviar el formulario con cada dato inválido por separado y todos juntos;
verificar los mensajes, que los datos se conservan y que no se crea ningún proyecto. Simular una
falla de guardado y verificar el mensaje general.

**Acceptance Scenarios**:

1. **Given** el formulario de creación con el nombre vacío, fechas inconsistentes o sin
   integrantes, **When** presiono "Crear", **Then** veo los mensajes exactos de HU-01 (nombre
   obligatorio, fecha de fin anterior a la de inicio, al menos un integrante), cada uno junto a su
   campo, los datos ingresados se conservan y no se crea ningún proyecto.
2. **Given** que dejo vacía la fecha de inicio o la de fin, **When** presiono "Crear", **Then** veo
   un mensaje junto a ese campo que indica cuál fecha falta y no se crea el proyecto.
3. **Given** que se envía una fecha con un formato inválido, **When** presiono "Crear", **Then**
   veo un mensaje junto a ese campo y no se crea el proyecto.
4. **Given** que se envía un integrante no numérico o inexistente, **When** presiono "Crear",
   **Then** veo un mensaje junto al campo de integrantes y no se crea el proyecto.
5. **Given** un formulario válido, **When** falla el guardado, **Then** veo "La operación no pudo
   completarse", los datos ingresados se conservan, no se muestra mensaje de éxito y el proyecto no
   aparece en la lista.

---

### User Story 3 - Ver la lista de proyectos (Priority: P2)

Como Usuario, quiero ver los proyectos existentes en la misma pantalla, para saber qué hay creado
antes de agregar uno nuevo.

**Why this priority**: Confirma visualmente el resultado del alta y orienta al equipo, pero no
bloquea la creación.

**Independent Test**: Con dos proyectos de distintas fechas y cantidad de integrantes, abrir la
pantalla y verificar columnas y orden; con la base sin proyectos, verificar el mensaje y el
formulario.

**Acceptance Scenarios**:

1. **Given** proyectos existentes, **When** abro "Gestión de Proyectos", **Then** veo cada proyecto
   con su nombre, fechas de inicio y fin y cantidad de integrantes, con los más recientes primero.
2. **Given** que no hay proyectos, **When** abro la pantalla, **Then** veo el mensaje "Aún no hay
   proyectos" y el formulario para crear el primero.
3. **Given** que falla la lectura de proyectos o de integrantes, **When** abro la pantalla,
   **Then** veo la página de error de la aplicación y no un formulario incompleto.

---

### Edge Cases

- Un nombre con solo espacios se trata como vacío (regla de HU-01; se verifica en T001).
- Los nombres de proyectos pueden repetirse: HU-01 no define unicidad.
- Si el formulario envía el mismo integrante más de una vez, se toma una sola vez.
- Si el envío no incluye alguno de los campos, se trata como vacío y se informa junto al campo.
- Las fechas se ingresan con el control de fecha del navegador (`AAAA-MM-DD`); cualquier otro
  formato se rechaza como formato inválido aunque el navegador no lo haya impedido.
- Un nombre con HTML o scripts se muestra como texto en la lista (regla de HU-20a/HU-20b).
- Crear un proyecto no lo selecciona como proyecto activo (HU-20b: sin autoselección).
- Un nombre de más de 255 caracteres se rechaza con un mensaje junto al campo en lugar de fallar
  al guardar (extra, ver T027).
- La lista no tiene paginación; con muchos proyectos se muestra completa.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST ofrecer el acceso al formulario de alta desde la sección "Gestión de
  Proyectos" del menú.
- **FR-002**: El formulario MUST pedir nombre, integrantes (selección múltiple) y fechas de inicio
  y de fin.
- **FR-003**: El selector de integrantes MUST ofrecer solo integrantes existentes en el sistema.
- **FR-004**: Cuando no existan integrantes, el sistema MUST mostrar "No hay integrantes
  disponibles" y MUST NOT permitir crear el proyecto.
- **FR-005**: Un alta válida MUST crear el proyecto según las reglas de HU-01, mostrar el mensaje
  de éxito una sola vez y mostrar el proyecto en la lista.
- **FR-006**: Ante datos inválidos, el sistema MUST mostrar los mensajes de HU-01 junto a cada
  campo, conservar los datos ingresados y MUST NOT crear el proyecto.
- **FR-007**: Cuando falte la fecha de inicio o la de fin, el sistema MUST indicar cuál falta.
- **FR-008**: El sistema MUST rechazar fechas con formato inválido con un mensaje junto al campo.
- **FR-009**: El sistema MUST rechazar integrantes no numéricos o inexistentes con un mensaje
  junto al campo de integrantes.
- **FR-010**: El sistema MUST permitir la misma fecha de inicio y de fin.
- **FR-011**: La lista MUST mostrar nombre, fechas de inicio y fin y cantidad de integrantes de
  cada proyecto, con los más recientes primero.
- **FR-012**: Cuando no existan proyectos, el sistema MUST mostrar "Aún no hay proyectos" y el
  formulario de alta.
- **FR-013**: Ante una falla de guardado, el sistema MUST mostrar "La operación no pudo
  completarse", conservar los datos, MUST NOT mostrar éxito y MUST NOT mostrar el proyecto en la
  lista.
- **FR-014**: Recargar la página después de crear un proyecto MUST NOT crear un segundo proyecto.
- **FR-015**: Crear un proyecto MUST NOT cambiar el proyecto activo.
- **FR-016**: Si falla la lectura de proyectos o integrantes, el sistema MUST mostrar la página de
  error de la aplicación y MUST NOT mostrar un formulario incompleto.
- **FR-017**: Los mensajes de error MUST NOT exponer detalles internos de la base de datos.

### Key Entities *(include if feature involves data)*

- **Proyecto**: Definido por HU-01; esta historia lo crea desde el formulario y lo lista.
- **Integrante disponible**: Integrante existente en el sistema que puede asignarse a un proyecto.
- **Fila de la lista**: Nombre, fechas y cantidad de integrantes de un proyecto.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de las altas válidas aparece en la lista con los datos ingresados.
- **SC-002**: El 100% de las entradas inválidas se rechaza con el mensaje junto a su campo,
  conservando los datos y sin crear proyectos.
- **SC-003**: Al menos el 90% de las personas de prueba completa el alta sin ayuda en el primer
  intento. *(Se verifica manualmente en la Sprint Review; confirmar el umbral contra la spec de
  HU-01.)*
- **SC-004**: El tiempo promedio para completar el alta es menor a 2 minutos. *(Se verifica
  manualmente en la Sprint Review; confirmar el umbral contra la spec de HU-01.)*
- **SC-005**: El 100% de las fallas de guardado muestra "La operación no pudo completarse", sin
  éxito y sin proyecto en la lista.
- **SC-006**: La pantalla responde en menos de 500 ms bajo condiciones normales de uso.
- **SC-007**: El 100% de las recargas posteriores a un alta no crea proyectos duplicados.

## Assumptions

- El modelo, las validaciones, el caso de uso de alta y la persistencia de proyectos son los de
  HU-01; esta historia no cambia sus reglas ni sus mensajes.
- Los integrantes ya existen en el sistema: la tabla de integrantes se alimenta fuera de HU-01. En
  el Sprint 2 se cargan con un script semilla (`db/seeds/members.sql`), que no es una migración.
  El alta de integrantes desde la aplicación queda como historia nueva del backlog.
- El estado de un proyecto (No iniciado, En curso, Finalizado) es de HU-12; esta lista no lo
  muestra.
- Mensajes, formularios y protección contra envíos ajenos vienen de HU-20b (US1 y US2); la
  estructura común y los errores, de HU-20a.
- Seleccionar el proyecto activo se hace desde la pantalla de HU-20b; esta lista no incluye el
  botón de selección ni reutiliza esa pantalla.
- Modificar proyectos, quitar integrantes, paginación y filtros quedan fuera del alcance.
- No hay roles: cualquier persona que use la aplicación puede crear proyectos.
