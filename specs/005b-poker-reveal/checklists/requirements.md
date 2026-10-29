# Checklist de calidad de la especificación: HU-05b

**Propósito**: validar que la especificación está completa antes de implementar.
**Creada**: 2026-10-28
**Feature**: [spec.md](../spec.md)

## Calidad del contenido

- [x] La historia sigue el formato "Como…, quiero…, para…".
- [x] Los escenarios no contienen detalles de implementación.
- [x] Enfocada en el valor para el Scrum Master (guiar la discusión hacia un consenso).
- [x] Todas las secciones obligatorias están completas (Sección SDD: intención, alcance, fuera de alcance,
  entradas, salidas, reglas, restricciones, condiciones de error).

## Completitud de requisitos

- [x] No quedan marcadores [NEEDS CLARIFICATION], "(a definir)" ni preguntas abiertas (1 duda respondida el 28/10,
  ver "Clarifications").
- [x] Los 12 escenarios del Issue están en Dado/Cuando/Entonces simples, sin "Esquema del escenario".
- [x] Los datos de ejemplo son los del Issue y cierran con la serie (niveles, diferencias y límite de dos niveles).
- [x] Cada escenario tiene al menos un requisito funcional que lo cubre.
- [x] Cada requisito es verificable y usa MUST/MUST NOT (FR-001 a FR-019).
- [x] Los criterios de éxito son medibles (SC-001 a SC-007).
- [x] Los casos límite están identificados (identificadores inválidos, otro proyecto, orden 404 antes de 409,
  cierres simultáneos, voto concurrente, integrante quitado, ronda cerrada sin votos, extremos de la serie, escape,
  recarga).
- [x] Alcance y fuera de alcance definidos (nueva ronda y estimación acordada, roles, cierre automático, reabrir).
- [x] Dependencias explícitas: HU-05a, HU-20a, HU-20b; habilita HU-18.
- [x] Los códigos 303/409/404/500 están definidos por caso.
- [x] Los mensajes están definidos en un solo lugar (el archivo de mensajes de HU-05a).

## Preparación

- [x] La tabla de trazabilidad conecta escenario, requisito, éxito y prueba, con los mismos nombres de prueba que
  `tasks.md`.
- [x] Todos los requisitos y criterios de éxito tienen tareas en `tasks.md` (tabla "requisitos → tareas"), con 30
  tareas como máximo.
- [x] El plan respeta la constitución v1.1.0: el método nuevo del puerto `poker.Store` y la lectura nueva de
  `web.PokerCatalog` tienen adaptador PostgreSQL y pruebas de integración; RED → GREEN por bloque con
  `evidencia-red.txt`.
- [x] Sin migración nueva y sin editar migraciones existentes.
- [x] Solo se editan archivos de HU-05a; sus pruebas siguen en verde.
- [x] La revelación se limita por construcción a rondas cerradas y una prueba verifica que un voto de una ronda
  abierta nunca aparece.
- [ ] Aceptada por el equipo (completar al subir el Issue).

## Notas

- Decisiones del equipo registradas en "Clarifications" (sesión 2026-10-28).
- `/speckit-clarify` (28/10): sin preguntas nuevas; la única duda real (voto de un integrante quitado del proyecto)
  se respondió durante `/speckit-specify`: se muestra y cuenta para el rango.
- `/speckit-analyze` (28/10): sin problemas críticos ni violaciones de la constitución; cobertura de FR y SC 100%
  (19 FR y 7 SC, 30 tareas). Se corrigieron:
  - I1, en `plan.md`: `waitForWaiters` va en `reveal_fixtures_integration_test.go`, como dice T008;
  - I2, en `data-model.md`, `plan.md`, T015, T017, T018 y T024: "Rondas cerradas" va al final de la página y las
    pruebas de HU-05b buscan cada fila dentro de su ronda (`roundRow`, `closedRow`), porque `data-integrante` se
    repite entre secciones y `participantRow` de HU-05a toma la primera coincidencia;
  - I3, en T019: en el BDD del escenario 7, Ana cierra «Login» desde la pantalla (el escenario dice "cerrada por Ana");
  - U1, en "Supuestos": el proyecto «Beta» (integrante Dani) no está en el Issue y solo se usa para el escenario 10;
  - C1, en la trazabilidad: `TestPostgresCloseRound_EsperaAlVotoEnCurso` pasa del escenario 6 a su propio caso límite
    (FR-012);
  - L1, en "Clarifications" y "Restricciones": se aclara que la lista revelada se amplía en la sesión siguiente y que
    también se crean archivos nuevos.
  - Queda documentado sin prueba propia: FR-014 se cubre con `TestHU05a_ErrorAlLeer` (las consultas nuevas usan el
    mismo `pokerEach` con `Unavailable`).
- Revisión de cuentas: los cinco casos del Issue cierran con la serie (Login 2→6 = 4, Pagos 3→5 = 2, Reportes 3→6 = 3,
  Perfil 4→4 = 0, Ayuda sin votos) y coinciden con las notas del equipo (3-8 no, 3-13 sí, 2-13 sí).
- Solo queda sin marcar "Aceptada por el equipo", que se completa al subir el Issue.
