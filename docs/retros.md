# Actas del Equipo

Registro de reuniones de seguimiento (dailies informales), Sprint Reviews y Retrospectivas
del proyecto **Software Metrics & Estimation**.

---

## Sprint 1

**Sprint Goal:** Dejar funcionando el ciclo básico de gestión: crear un proyecto, cargarle
historias al backlog, organizarlas en un sprint, y que el sistema empiece a calcular
métricas de seguimiento.

**Historias:** HU-01, HU-02, HU-04, HU-07 (planificadas) y HU-03 (incorporada durante el Sprint)

**Fecha de cierre:** 18 de octubre de 2026

### Seguimiento (dailies informales)

Las dailies del Sprint 1 fueron informales y no se registraron por escrito. A partir del
Sprint 2 se anota una línea por sesión de trabajo (qué se hizo, qué sigue, bloqueos).

---

### Sprint Review

- **Fecha:** 18 de octubre de 2026
- **Historias completadas:**
  - HU-01 Alta de Nuevo Proyecto (3 SP)
  - HU-02 Creación de Historia en el Backlog (5 SP)
  - HU-03 Registro de horas trabajadas (3 SP, incorporada durante el Sprint)
  - HU-04 Creación y Planificación de Sprint (8 SP)
  - HU-07 Cálculo de Velocidad y Desviación de Esfuerzo (5 SP)
  - Total: 24 SP completados sobre 21 SP planificados al inicio.
- **Historias no completadas (y por qué):** ninguna de las comprometidas. Quedaron fuera de este
  Sprint las historias que se detectaron como faltantes al contrastar el backlog con el PDF de
  la cátedra (HU-11 a HU-25: cierre de Sprint, estado de historias, consulta de Sprints
  anteriores, horas estimadas, interfaz web, entre otras). Se planifican entre los Sprints 2 y 4.
- **Resultado frente al Sprint Goal:** el núcleo de reglas de negocio está implementado y probado
  contra PostgreSQL real (proyectos, backlog, esfuerzo, Sprints y métricas). El MVP completo
  no se alcanzó en este Sprint porque faltaban historias en el backlog; se alcanza con el
  Sprint 2.
- **Feedback del Product Architect:** (completar luego de la presentación)

---

### Retrospectiva

- **Fecha:** 18 de octubre de 2026
- **Qué funcionó bien:**
  - Especificar cada historia con SDD, BDD y TDD antes de programar dejó pruebas reproducibles
    y evidencia por historia.
  - Probar contra PostgreSQL real, y no contra datos simulados, detectó problemas que de otro
    modo se habrían visto recién al integrar.
  - La base de datos se decidió con una encuesta rápida del equipo y se mantuvo.
  - Contrastar el backlog con el PDF de la cátedra permitió detectar las historias faltantes
    antes de cerrar el Proyecto.
  - Reasignar HU-03 a la Scrum Master destrabó la dependencia de HU-07.
- **Qué no funcionó / mejorar:**
  - El backlog estaba incompleto respecto del PDF: el MVP no estaba cubierto y se detectó recién
    al cerrar el Sprint.
  - Las dependencias entre historias no se analizaron al planificar: HU-07 dependía de HU-03.
  - Las migraciones se numeraron por número de historia y se aplicaron fuera de orden:
    `migrate up` no aplica una migración de número menor a la versión ya aplicada, y la 003 se
    salteó en una base.
  - Copiar archivos de otras historias entre ramas para poder probar generó conflictos de
    merge en el PR de HU-07 (unos 20 archivos).
  - Una prueba de integración de HU-01 se rompió al existir tablas que referencian a sus
    tablas, y las pruebas de integración corridas en paralelo se pisan porque comparten una
    única base.
  - Documentación desactualizada: el `tasks.md` de HU-07 tenía la carpeta y los números de
    migración viejos, y el README tenía el orden de migraciones viejo.
  - El cierre del Sprint se movió del 9 al 18 de octubre.
- **Acciones para el próximo sprint:**
  1. Una migración nueva toma el último número de `main` más uno, sin importar el número de su
     historia, y se agrega después de que sus dependencias están en `main`.
  2. Mergear los PR con "Create a merge commit" y hacer `git merge origin/main` seguido en cada
     rama, en lugar de copiar archivos de otras historias.
  3. Correr las pruebas con `go test -count=1 -p 1 ./...` y dejarlo escrito en el README y en
     los quickstarts.
  4. Revisar la matriz de dependencias antes de cada planning y marcar las dependencias en cada
     Issue ("Depende de").
  5. Verificar con up, down y up desde cero la migración `down` de HU-04.
  6. Corregir en el README el orden y los dueños de las migraciones.
  7. Agregar un `.gitattributes` para unificar los saltos de línea y que `gofmt -l` no marque
     archivos por el formato de Windows.
  8. Registrar las dailies en este archivo, una línea por sesión.
  9. Hacer el Planning Poker del Sprint 2 incluyendo HU-20a y HU-20b y recalcular la capacidad.