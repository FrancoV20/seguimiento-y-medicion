# Checklist de calidad de la especificación: HU-28

**Propósito**: validar que la especificación está completa antes de implementar.
**Creada**: 2026-10-28
**Feature**: [spec.md](../spec.md)

## Calidad del contenido

- [x] La historia sigue el formato "Como…, quiero…, para…".
- [x] Los escenarios no contienen detalles de implementación.
- [x] Enfocada en el valor para el Scrum Master (las decisiones de planificación no las toma cualquiera por error).
- [x] Todas las secciones obligatorias están completas (Sección SDD: intención, alcance, fuera de alcance,
  entradas, salidas, reglas, restricciones, condiciones de error).

## Completitud de requisitos

- [x] No quedan marcadores [NEEDS CLARIFICATION], "(a definir)" ni preguntas abiertas.
- [x] Los 13 escenarios (12 del Issue y el 13 agregado por el equipo) están en Dado/Cuando/Entonces simples.
- [x] Los datos de ejemplo son los del Issue; lo agregado para las pruebas (contraseñas, «Perfil» y «Reportes») está
  marcado como agregado.
- [x] Cada escenario tiene al menos un requisito funcional que lo cubre.
- [x] Cada requisito es verificable y usa MUST/MUST NOT (FR-001 a FR-017).
- [x] Los criterios de éxito son medibles (SC-001 a SC-006).
- [x] Los casos límite están identificados (varios Scrum Master, cambio de rol con sesión abierta, proyectos sin Scrum
  Master al migrar, HU-11 quita al único, secciones sin `RequiresMember`, otro origen, `GET`, comando con rol,
  integrante, opciones o entorno inválidos).
- [x] Alcance y fuera de alcance definidos (Product Owner, pantalla de roles, botones, permisos por sección, acciones
  que siguen libres).
- [x] Dependencias explícitas: cuáles están en `main` y cuáles se toman de sus documentos con "(T001)".
- [x] Los códigos 303/403/500/503 están definidos por caso y el orden del control está fijado.
- [x] Los textos de la pantalla y del comando se definen en un solo lugar.

## Preparación

- [x] La tabla de trazabilidad conecta escenario, requisito, éxito y prueba, con los mismos nombres de prueba que
  `tasks.md`.
- [x] Todos los requisitos y criterios de éxito tienen tareas en `tasks.md` (tabla "requisitos → tareas"), con 35
  tareas como máximo.
- [x] El plan respeta la constitución v1.1.0: puertos con adaptador PostgreSQL y pruebas de integración; RED →
  GREEN por bloque con `evidencia-red.txt`.
- [x] La migración nueva tiene up y down, con prueba, y su número se confirma en T001 y antes del PR.
- [ ] Aceptada por el equipo (completar al subir el Issue).

## Notas

- Las decisiones del equipo están registradas en "Clarifications" (sesión 2026-10-28).
- `/speckit-clarify` (28/10): una pregunta. Si el `{id}` de `POST /proyectos/{id}/modificar` no es el proyecto activo,
  403 (FR-016, regla 1 y casos límite).
- Corrección del equipo (28/10) sobre FR-016: el rechazo por otro proyecto usa un mensaje propio,
  `MsgScrumMasterOtherProject` ("Para modificar este proyecto, elegilo como proyecto activo e identificate como su
  Scrum Master"), sin consultar el rol en otro proyecto. Se agregó el escenario 13 y se actualizaron:
  - en spec.md: FR-016, regla 1, condiciones de error, casos límite, SC-001 y la trazabilidad (fila 13; la fila 5 ya
    no cita la prueba de otro proyecto);
  - en data-model.md: la fila 2b y los textos;
  - en research.md: la decisión 5;
  - en tasks.md: T018 (`TestRequireScrumMasterOfPathProject_OtroProyecto`), T020 (`TestHU28_ModificarOtroProyecto`)
    y T035;
  - quickstart.md y plan.md.

  La tabla requisitos → tareas no cambia (FR-016 ya estaba en T018, T020, T022 y T023).
- `/speckit-analyze` (28/10): sin problemas críticos ni violaciones de la constitución. Cobertura de FR y SC del 100%
  (17 FR, 6 SC, 35 tareas). Se corrigieron:
  - U1, en la trazabilidad de spec.md: el escenario 12 citaba una prueba del 403 para un 503; ahora cita la prueba de
    HU-20b que cubre la página sin detalles;
  - C1, en T020: FR-012 (los botones siguen visibles) no tenía prueba; se suma el `GET /sprints` de Beto;
  - I1, en la estructura de plan.md: faltaba `src/web/member_role_context_test.go` (T014);
  - I2, en "Datos de ejemplo" de spec.md: el Sprint 1 finalizado de «Alfa», que usa data-model.md, no estaba marcado
    como agregado.

  Quedan documentados:
  - `<NNN>` (número de la migración): marcador intencional que fija T001 y reconfirma T033;
  - las rutas de HU-11, HU-15, HU-05b y HU-18 que no estén en `main` se protegen en la rama de esa historia (T001,
    T003);
  - la decisión del equipo dice `run(args, stdout, getenv)`; se agrega `ctx`, como en `set-password` (research.md,
    decisión 9).
- Hallazgos de la verificación de nombres:
  - HU-11 agrega `ErrProjectNotFound` a `src/project`; por eso los roles van en un paquete nuevo, `src/role`;
  - HU-23 prueba con dobles de `ContextStore` sin integrante; el Bloque E les agrega un Scrum Master con cookie
    firmada;
  - hoy son 5 los archivos de prueba que hacen POST a rutas protegidas.
- Solo queda sin marcar "Aceptada por el equipo", que se completa al subir el Issue.
