# Checklist de calidad de la especificación: HU-08

**Propósito**: validar que la especificación está completa antes de implementar.
**Creada**: 2026-10-28
**Feature**: [spec.md](../spec.md)

## Calidad del contenido

- [x] La historia sigue el formato "Como…, quiero…, para…".
- [x] Los escenarios no contienen detalles de implementación.
- [x] Enfocada en el valor para el Scrum Master.
- [x] Todas las secciones obligatorias están completas (Sección SDD: intención, alcance, fuera de alcance,
  entradas, salidas, reglas, restricciones, condiciones de error).

## Completitud de requisitos

- [x] No quedan marcadores [NEEDS CLARIFICATION], "(a definir)" ni preguntas abiertas.
- [x] Los 12 escenarios del Issue están en Dado/Cuando/Entonces simples, sin "Esquema del escenario".
- [x] Los datos de ejemplo cierran aritméticamente (cantidades, Story Points, acumulada, promedio, desviación,
  defectos y alturas de barras) y ningún Sprint se usa para dos repartos distintos.
- [x] Cada escenario tiene al menos un requisito funcional que lo cubre.
- [x] Cada requisito es verificable y usa MUST/MUST NOT (FR-001 a FR-018).
- [x] Los criterios de éxito son medibles (SC-001 a SC-007).
- [x] Los casos límite están identificados (Sprint sin historias, más de 10 Sprints, redondeos, barras mínimas,
  desviación negativa y cero, Sprint sin registro de cierre, rechazo de HU-07, otro proyecto, recarga).
- [x] Alcance y fuera de alcance definidos (HU-09, HU-16, filtros, página de inicio, defectos por severidad).
- [x] Dependencias explícitas: HU-06, HU-07, HU-10, HU-14, HU-15, HU-20a, HU-20b; orden de implementación después de
  HU-10.
- [x] Los códigos 200/303/405/500/503 están definidos por caso.
- [x] Los textos de la pantalla se definen en un solo lugar.

## Preparación

- [x] La tabla de trazabilidad conecta escenario, requisito, éxito y prueba, con los mismos nombres de prueba que
  `tasks.md`.
- [x] Todos los requisitos y criterios de éxito tienen tareas en `tasks.md` (tabla "requisitos → tareas").
- [x] El plan respeta la constitución v1.1.0: puerto con adaptador PostgreSQL y pruebas de integración; RED →
  GREEN por bloque con `evidencia-red.txt`.
- [x] Sin migración nueva y sin editar migraciones existentes.
- [x] Los nombres de HU-06, HU-10, HU-15 y HU-25 que todavía no están en `main` quedan marcados para confirmar en
  T001.
- [ ] Aceptada por el equipo (completar al subir el Issue).

## Notas

- Las decisiones del equipo están registradas en "Clarifications" (sesión 2026-10-28).
- `/speckit-clarify` (28/10): una pregunta. Se muestra el aviso de cálculo parcial de HU-07 tal cual (FR-007).
- `/speckit-analyze` (28/10) no encontró problemas críticos. Se corrigieron:
  - I1, en plan.md: T038 → T035;
  - I2, en la trazabilidad: la revisión sin persistencia es T029;
  - U1, en la trazabilidad: `TestHU08_SprintSinHistorias` y `TestNewDashboardChart`;
  - C1, en la tabla requisitos → tareas: FR-016 suma T025;
  - I3, en Supuestos: los escenarios con cantidades agregadas son el 3, 4, 6 y 8;
  - F1, en tasks.md: T022 sin [P];
  - L1, en los escenarios: conservan el número del Issue.
- Revisión de cuentas: los datos de ejemplo, los casos límite y las fixtures de T009 y T022 cierran. Cada proyecto
  de prueba usa Sprints propios. En «Parcial», H tiene Story Points y no tiene horas estimadas.
- Solo queda sin marcar "Aceptada por el equipo", que se completa al subir el Issue.
