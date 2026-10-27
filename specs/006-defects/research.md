# Research: Registro y trazabilidad de defectos

## Decision 1: Paquete de dominio nuevo y pantalla propia

- **Decision**: Reglas y servicio en `src/defect`; pantalla `/defectos` con formulario y lista.
- **Rationale**: Los defectos son una entidad nueva con reglas propias; el paquete propio evita editar archivos de
  HU-02 o HU-04. La pantalla sigue el criterio del equipo de entregar cada historia con su interfaz.

## Decision 2: Sin Sprint activo se bloquea el registro

- **Decision**: Sin Sprint activo no se registra y la pantalla no muestra el formulario.
- **Rationale**: El criterio original dice que el defecto queda "asociado al Sprint actual como Sprint de
  detección"; sin Sprint actual no hay valor para esa columna. Es la misma decisión del equipo para HU-19 (resolver
  sin Sprint activo se bloquea). Para HU-06 se confirma.
- **Alternatives considered**: Elegir el Sprint de detección a mano (descartada: permite datos falsos y complica
  HU-10); permitir el defecto sin Sprint (descartada: rompe el conteo por Sprint).

## Decision 3: Niveles de severidad

- **Decision**: Crítica, Alta, Media y Baja, definidos en un solo lugar del dominio.
- **Rationale**: El enunciado pide severidad sin fijar niveles; cuatro niveles es una escala habitual y se puede
  cambiar sin tocar otra cosa. Propuesta del equipo a confirmar.
- **Alternatives considered**: Tres niveles (descartada: menos resolución para el dashboard); texto libre
  (descartada: no se puede agrupar ni validar).

## Decision 4: Historia relacionada de cualquier estado

- **Decision**: Se acepta cualquier historia del proyecto activo, sin importar su estado ni su Sprint.
- **Rationale**: Un defecto suele aparecer después de que la historia se terminó. Si solo se aceptaran las del
  Sprint activo, no se podrían registrar defectos de historias "Hecho" de Sprints anteriores.
- **Alternatives considered**: Solo historias del Sprint activo (descartada: pierde defectos reales).

## Decision 5: Registro atómico y bloqueo contra el cierre

- **Decision**: Una transacción que toma el Sprint activo con `FOR SHARE`, comprueba la historia e inserta el
  defecto.
- **Rationale**: El cierre de HU-15 bloquea el Sprint con `FOR UPDATE`; `FOR SHARE` hace que un registro y un cierre
  simultáneos se serialicen: el defecto queda en el Sprint antes del cierre, o se rechaza porque ya no hay Sprint
  activo. Nunca queda un Sprint de detección incoherente.
- **Alternatives considered**: Leer el Sprint sin bloquear (descartada: carrera con el cierre); `FOR UPDATE`
  (descartada: serializaría registros entre sí sin necesidad).

## Decision 6: Validación completa en el dominio

- **Decision**: `Validate` devuelve todas las violaciones juntas; el servicio las usa antes de guardar y la web las
  traduce a mensajes por campo.
- **Rationale**: Mostrar los errores uno por uno obliga a reenviar el formulario varias veces; es el mismo criterio
  que en HU-24.
- **Alternatives considered**: Validar en el handler (descartada: duplica reglas).

## Decision 7: Restricciones también en la base

- **Decision**: La tabla tiene `CHECK` de severidad, estado y coherencia entre estado y Sprint de resolución, y la
  descripción no vacía.
- **Rationale**: Una segunda barrera ante datos inválidos y la base ya deja listo el modelo para HU-19 (un defecto
  "Resuelto" siempre tiene Sprint de resolución y uno "Abierto" nunca).
- **Alternatives considered**: Solo validar en Go (descartada: no protege ante otros escritores).

## Decision 8: Lista simple, más recientes primero

- **Decision**: La lista muestra todos los defectos del proyecto ordenados por identificador descendente, sin
  paginación ni filtros.
- **Rationale**: El volumen esperado es pequeño y HU-10 y HU-08 resumen los datos; los filtros no los pide ninguna
  historia.
- **Alternatives considered**: Filtros por estado o severidad (descartada: no están en la historia).

## Decision 9: Sin identificación de quien reporta

- **Decision**: La sección requiere proyecto pero no integrante, y el defecto no guarda quién lo reportó.
- **Rationale**: El enunciado no incluye ese dato entre los mínimos de un defecto y la autenticación se resuelve
  después del Sprint 2.
- **Alternatives considered**: Exigir integrante identificado (descartada: agrega una columna y una regla que ninguna
  historia pide).

## Decision 10: Códigos de respuesta

- **Decision**: Registro válido: 303. Datos inválidos o historia ajena: 422. Sin Sprint activo: 409. Falla de
  guardado: 500. Falla de lectura: `Fail` de HU-20a. Sin proyecto: 303 de HU-20b.
- **Rationale**: Distingue los casos para las pruebas y el navegador.

## Decision 11: Estrategia de pruebas

- **Decision**: Pruebas del servicio, del adaptador y del catálogo contra PostgreSQL real con fixtures propias
  (dos proyectos; Sprint activo y Sprint finalizado; historias de distintos estados); una prueba de concurrencia
  entre el registro y el cierre; un almacenamiento y un catálogo que fallan solo para simular las fallas de
  guardado y de lectura; BDD.
- **Nota de fixtures**: `user_stories.project_id` no tiene borrado en cascada: se borran primero las historias y
  después el proyecto. La tabla `defects` cae en cascada con las historias, los Sprints y el proyecto.