# Feature Specification: Formularios seguros y contexto de trabajo

**Feature Branch**: `020b-project-context`

**Created**: 2026-10-17 | **Updated**: 2026-10-18

**Status**: Implemented

**Input**: User description: "HU-20b / Formularios Seguros y Contexto de Trabajo: Como Usuario, quiero que los formularios y mensajes funcionen igual en todo el sistema sin perder mis datos, y poder seleccionar un proyecto e identificarme como integrante de ese proyecto, para que las secciones muestren solo los datos del proyecto y mis acciones (esfuerzo, votos) queden asociadas a mi nombre. Depende de HU-20a (base web) y de los datos de HU-01."

> **División de HU-20**: HU-20a construye el servidor, la estructura común y los errores. HU-20b
> (esta) construye los mensajes y formularios, la protección contra envíos ajenos y el contexto
> de trabajo. Orden recomendado: primero US1 y US2 (con eso HU-21 ya puede empezar) y después US3
> a US5 (que necesitan HU-22 a HU-25).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Mensajes y formularios con comportamiento consistente (Priority: P1)

Como Usuario, quiero que los mensajes de éxito, error y advertencia y los errores de formulario
se vean y se comporten igual en todas las pantallas, sin perder lo que escribí, para entender
rápido qué pasó y corregirlo.

**Why this priority**: HU-21 a HU-25 repiten estas reglas; definirlas acá evita incoherencias y
trabajo duplicado, y es lo que permite que HU-21 empiece.

**Independent Test**: Con un formulario de prueba enviado con datos inválidos, válidos y con
falla de guardado, y recargando la página resultante.

**Acceptance Scenarios**:

1. **Given** cualquier pantalla, **When** muestra un mensaje de éxito, error o advertencia,
   **Then** cada tipo tiene su propio formato y una etiqueta o ícono de texto además del color,
   y es el mismo en todas las pantallas.
2. **Given** un formulario con datos inválidos, **When** lo envío, **Then** cada error aparece
   junto a su campo y los datos ingresados se conservan.
3. **Given** un formulario válido, **When** falla el guardado, **Then** veo "La operación no
   pudo completarse", los datos ingresados se conservan y no se muestra éxito.
4. **Given** que envío un formulario válido, **When** el sistema responde, **Then** me lleva a
   una página de resultado con el mensaje de éxito, y si recargo la página no se reenvía el
   formulario ni se duplica el registro.
5. **Given** que se mostró un mensaje de éxito, **When** navego a otra pantalla o recargo,
   **Then** el mensaje no vuelve a aparecer.

---

### User Story 2 - Mostrar contenido de forma segura y aceptar envíos solo de la aplicación (Priority: P1)

Como Usuario, quiero que lo que se escribe en la aplicación nunca se ejecute como código y que
los formularios solo se acepten desde la propia aplicación, para proteger al equipo y a sus datos.

**Why this priority**: Es seguridad básica de una aplicación web y es más barato resolverla una
vez en la base que en cada formulario.

**Independent Test**: Mostrar texto con etiquetas HTML y enviar formularios desde el mismo
origen y desde un origen distinto.

**Acceptance Scenarios**:

1. **Given** un texto con HTML o scripts (por ejemplo, `<script>…</script>`), **When** se muestra
   en cualquier pantalla, **Then** se ve como texto y no se ejecuta.
2. **Given** un formulario enviado desde otro sitio web, **When** llega al sistema, **Then** se
   rechaza con código 403 y un mensaje claro, sin ejecutar ninguna acción.
3. **Given** un formulario enviado desde la propia aplicación, **When** llega al sistema,
   **Then** se procesa normalmente.

---

### User Story 3 - Seleccionar el proyecto activo (Priority: P2)

Como Usuario, quiero elegir con qué proyecto trabajo y que esa elección se mantenga mientras
navego, para ver solo los datos de ese proyecto sin volver a elegirlo en cada pantalla.

**Why this priority**: HU-14, HU-22, HU-23, HU-24 y HU-25 muestran datos de un proyecto; sin
proyecto activo no pueden mostrar nada.

**Independent Test**: Con dos proyectos existentes, seleccionar uno, navegar entre secciones,
cambiar de proyecto y repetir con identificadores inválidos y con un proyecto que desaparece.

**Acceptance Scenarios**:

1. **Given** que existen proyectos, **When** selecciono uno y navego entre secciones, **Then**
   las secciones muestran únicamente los datos de ese proyecto sin volver a elegirlo, y su nombre
   queda visible en la estructura común.
2. **Given** que no existen proyectos, **When** abro la aplicación, **Then** veo el mensaje "Aún
   no hay proyectos", y el acceso para crear el primero aparece cuando exista la sección de
   Proyectos (HU-21).
3. **Given** un proyecto seleccionado con un integrante identificado, **When** cambio a otro
   proyecto, **Then** la identificación del integrante se limpia.
4. **Given** un proyecto seleccionado con un integrante identificado, **When** vuelvo a
   seleccionar el mismo proyecto, **Then** la identificación del integrante se mantiene.
5. **Given** un identificador de proyecto inexistente o inválido, **When** intento
   seleccionarlo, **Then** veo un mensaje claro, la aplicación no falla y se conserva la
   selección anterior si la había.
6. **Given** que el proyecto que tenía seleccionado ya no existe en el sistema, **When** vuelvo
   a usar la aplicación, **Then** la selección se limpia, veo una advertencia y se me pide
   elegir otro proyecto.
7. **Given** que existe un solo proyecto, **When** abro la aplicación, **Then** no se
   selecciona automáticamente y se me pide elegirlo.
8. **Given** un proyecto seleccionado, **When** cierro el navegador y vuelvo a abrir la
   aplicación, **Then** se me pide seleccionar el proyecto nuevamente.

---

### User Story 4 - Identificarme como integrante del proyecto (Priority: P2)

Como integrante del equipo, quiero elegir mi nombre entre los integrantes del proyecto, para que
mis acciones (cargar esfuerzo, votar) queden registradas a mi nombre sin tener que escribirlo.

**Why this priority**: HU-24 (esfuerzo) y las historias de Planning Poker necesitan saber quién
actúa. Sin contraseña, la identificación es una elección dentro del equipo.

**Independent Test**: Con un proyecto con varios integrantes: identificarse, intentar una acción
que requiere integrante sin estarlo, y repetir con un integrante que no pertenece al proyecto o
que fue quitado.

**Acceptance Scenarios**:

1. **Given** un proyecto seleccionado, **When** abro el selector de integrante y elijo mi nombre,
   **Then** en el selector aparecen solo los integrantes de ese proyecto, el sistema usa mi
   nombre en mis acciones hasta que lo cambie (sin contraseña) y mi nombre queda visible en la
   estructura común mientras navego.
2. **Given** que no me identifiqué, **When** intento una acción que requiere integrante (cargar
   esfuerzo, votar), **Then** se me pide identificarme y luego vuelvo a donde estaba.
3. **Given** un integrante que no pertenece al proyecto activo, **When** se envía una acción en
   su nombre, **Then** se rechaza, se limpia la identificación y se me pide elegir un integrante
   válido.
4. **Given** que mi integrante identificado ya no pertenece al proyecto (por ejemplo, tras
   modificarlo), **When** navego por la aplicación, **Then** la identificación se limpia, veo una
   advertencia y se me pide elegir otro integrante.

---

### User Story 5 - Acceder a secciones que requieren contexto (Priority: P2)

Como Usuario, quiero que al entrar a una sección que necesita proyecto o integrante se me guíe
para elegirlos y luego vuelva a donde quería ir, para no perder el hilo de lo que estaba haciendo.

**Why this priority**: Evita pantallas vacías o errores cuando alguien llega a una dirección sin
haber elegido contexto, por ejemplo con un enlace guardado.

**Independent Test**: Entrar a una sección que requiere proyecto sin haberlo elegido, elegirlo y
verificar el regreso; y probar destinos de regreso inválidos. Mientras no existan las secciones
reales, se usan secciones de prueba.

**Acceptance Scenarios**:

1. **Given** que no hay proyecto seleccionado, **When** entro a una sección declarada como que
   requiere proyecto, **Then** se me redirige a la selección de proyecto con un mensaje y, al
   elegir uno, vuelvo a la sección que había pedido.
2. **Given** un destino de regreso externo o malformado, **When** elijo un proyecto o un
   integrante, **Then** el destino se ignora y voy a la página de inicio.
3. **Given** una sección que no requiere contexto (por ejemplo, Proyectos), **When** entro sin
   proyecto seleccionado, **Then** se muestra normalmente.

---

### Edge Cases

- Si un formulario envía más datos que el límite permitido, se rechaza con un error claro y no se
  procesa (extra opcional, ver Assumptions).
- Un mensaje de éxito o advertencia solo se muestra en la primera página que se carga después de
  la acción; una recarga posterior no lo repite.
- Si la cookie de proyecto o de integrante tiene un valor manipulado o no numérico, se trata como
  si no hubiera selección y se limpia, sin mostrar un error interno.
- Un identificador de proyecto o de integrante negativo o muy grande se trata igual que uno
  malformado.
- Si el integrante identificado pertenece a otro proyecto distinto del activo, se rechaza igual
  que un integrante inexistente.
- Si la identificación se pierde al enviar un formulario, los datos del formulario no se
  conservan: se pide identificarse y se vuelve a la pantalla de origen.
- Si el proyecto activo no puede validarse por indisponibilidad de la base de datos, se muestra la
  página de servicio no disponible de HU-20a y no se limpia la selección.
- Si la lista de proyectos o de integrantes está vacía, se muestra un mensaje en lugar de un
  selector vacío.
- Dos pestañas del mismo navegador comparten la selección: un cambio en una se refleja en la otra
  en su próxima solicitud.
- Seleccionar un proyecto o integrante mediante un envío de otro sitio se rechaza (protección de
  US2).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST mostrar los mensajes de éxito, error y advertencia con un formato
  propio por tipo, con una etiqueta o ícono de texto además del color, igual en todas las
  pantallas.
- **FR-002**: Los errores de validación de formularios MUST mostrarse junto a su campo y MUST
  conservar los datos ingresados.
- **FR-003**: Los fallos de guardado MUST informarse con el texto único "La operación no pudo
  completarse", MUST conservar los datos ingresados y MUST NOT mostrar mensaje de éxito.
- **FR-004**: Después de un envío válido, el sistema MUST llevar a una página de resultado; el
  mensaje de éxito MUST mostrarse una sola vez y recargar MUST NOT reenviar el formulario ni
  duplicar el registro.
- **FR-005**: El sistema MUST mostrar como texto todo contenido ingresado por usuarios, sin
  ejecutarlo.
- **FR-006**: El sistema MUST rechazar con 403 y un mensaje claro los envíos de formularios
  originados en otro sitio web, sin ejecutar ninguna acción, y MUST procesar con normalidad los
  originados en la propia aplicación.
- **FR-007**: El sistema MUST permitir seleccionar un proyecto existente como proyecto activo.
- **FR-008**: Las secciones MUST mostrar únicamente los datos del proyecto activo y el nombre del
  proyecto MUST quedar visible en la estructura común.
- **FR-009**: El proyecto activo MUST mantenerse al navegar entre secciones durante la sesión del
  navegador y MUST pedirse de nuevo al abrir una sesión nueva.
- **FR-010**: Cuando no existan proyectos, el sistema MUST mostrar "Aún no hay proyectos" y MUST
  ofrecer el acceso para crear el primero cuando exista la sección de Proyectos.
- **FR-011**: El sistema MUST NOT seleccionar un proyecto automáticamente, ni siquiera cuando
  exista uno solo.
- **FR-012**: Al cambiar a otro proyecto, el sistema MUST limpiar el integrante identificado; al
  volver a seleccionar el mismo proyecto, MUST conservarlo.
- **FR-013**: Ante un identificador de proyecto inexistente o inválido, el sistema MUST mostrar un
  mensaje claro, conservar la selección anterior y no fallar.
- **FR-014**: Si el proyecto activo deja de existir, el sistema MUST limpiar la selección, mostrar
  una advertencia y pedir elegir otro.
- **FR-015**: El sistema MUST permitir identificarse eligiendo entre los integrantes del proyecto
  activo, sin contraseña; el selector MUST mostrar solo integrantes de ese proyecto y el nombre
  elegido MUST quedar visible en la estructura común.
- **FR-016**: Una acción que requiere integrante MUST pedir identificarse si no hay integrante
  identificado y MUST volver a la pantalla de origen después de hacerlo.
- **FR-017**: El sistema MUST rechazar una acción de un integrante que no pertenece al proyecto
  activo, limpiar la identificación y pedir elegir uno válido.
- **FR-018**: Si el integrante identificado deja de pertenecer al proyecto, el sistema MUST limpiar
  la identificación, mostrar una advertencia y pedir elegir otro.
- **FR-019**: Al entrar a una sección que requiere proyecto sin tenerlo, el sistema MUST
  redirigir a la selección de proyecto con un mensaje y MUST volver a esa sección una vez elegido.
- **FR-020**: El sistema MUST ignorar destinos de regreso externos, malformados o que no
  correspondan a una sección propia, y MUST volver al inicio.
- **FR-021**: Una sección que no requiere contexto MUST mostrarse normalmente sin proyecto
  seleccionado.
- **FR-022**: El sistema MUST tratar valores manipulados o no numéricos de la selección como
  ausencia de selección, sin mostrar errores internos.
- **FR-023**: La selección y la identificación MUST validarse contra los datos actuales del
  sistema en cada solicitud.
- **FR-024**: Cuando no se pueda validar la selección por indisponibilidad de la base de datos, el
  sistema MUST mostrar el error de servicio no disponible y MUST NOT limpiar la selección.

### Key Entities *(include if feature involves data)*

- **Mensaje de aviso**: Texto de éxito, error o advertencia que se muestra una sola vez.
- **Error de formulario**: Mensaje asociado a un campo, junto con los valores ingresados.
- **Proyecto activo**: Proyecto elegido por la persona durante su sesión del navegador.
- **Integrante activo**: Integrante del proyecto activo con el que la persona realiza acciones.
- **Destino de regreso**: Sección a la que se vuelve después de elegir proyecto o integrante.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de los mensajes de éxito, error y advertencia usa el formato de su tipo y
  una etiqueta de texto, en las pruebas de todas las pantallas de prueba.
- **SC-002**: El 100% de los formularios enviados con datos inválidos conserva los datos
  ingresados y muestra cada error junto a su campo.
- **SC-003**: 0 registros duplicados al recargar la página que sigue a un envío válido.
- **SC-004**: El 100% de los textos con etiquetas HTML ingresados se muestra como texto y no se
  ejecuta.
- **SC-005**: El 100% de los envíos de formularios originados en otro sitio es rechazado.
- **SC-006**: El 100% de las secciones que requieren proyecto muestran solo datos del proyecto
  activo en las pruebas con dos proyectos con datos distintos.
- **SC-007**: El 100% de los accesos a secciones que requieren contexto sin tenerlo termina en la
  selección de proyecto o de integrante, y 0 en una pantalla vacía o un error interno.
- **SC-008**: El 100% de los valores de selección manipulados, no numéricos o inexistentes se
  trata como ausencia de selección y 0 produce error interno.
- **SC-009**: El 100% de las acciones con un integrante ajeno al proyecto activo es rechazado.
- **SC-010**: El 100% de los destinos de regreso externos o malformados es ignorado.
- **SC-011**: Elegir proyecto e integrante requiere como máximo 2 interacciones cada uno (abrir
  el selector y elegir).
- **SC-012**: La validación del contexto agrega menos de 50 ms a una solicitud bajo condiciones
  normales de uso.

## Assumptions

- La identificación es una comodidad dentro del equipo y no un mecanismo de seguridad: no hay
  contraseña y cualquier persona puede elegir cualquier integrante del proyecto. Es una decisión
  de alcance a confirmar con la cátedra.
- No se distinguen roles (Scrum Master, Product Builder) en esta historia; queda como deuda
  técnica documentada porque HU-01 no guarda el rol de cada integrante.
- La selección dura mientras dure la sesión del navegador; no hay opción de "recordarme".
- Los proyectos y sus integrantes ya existen por HU-01; esta historia solo los lee.
- La pantalla para crear proyectos es de HU-21; esta historia solo ofrece el acceso a ella cuando
  no hay proyectos y siempre que la sección esté registrada.
- La estructura común, los errores y las páginas 404, 405, 500 y 503 vienen de HU-20a.
- El texto de los mensajes de negocio lo define cada historia de dominio (HU-21 a HU-25); esta
  historia define su formato y el texto único "La operación no pudo completarse".
- Quitar integrantes de un proyecto es de HU-11; esta historia solo reacciona a que ya no
  pertenezcan al proyecto.
- Hasta que existan las secciones reales (Backlog, Sprints, Esfuerzo, Métricas), los escenarios
  sobre secciones que requieren proyecto o integrante se verifican con una sección de prueba.
- Extras fuera de los criterios de aceptación, a implementar solo si sobra tiempo: límite de
  tamaño del cuerpo de los formularios (413) y cabeceras de seguridad (política de contenido).
  Si el equipo los quiere obligatorios, hay que agregarlos como escenarios.
- Sincronizar la selección entre varios dispositivos y los permisos por rol quedan fuera del
  alcance de esta historia.
