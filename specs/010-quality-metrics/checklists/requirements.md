# Checklist de calidad de la especificación: HU-10

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
- [x] Los 12 escenarios del Issue están en Dado/Cuando/Entonces simples, sin "Esquema del escenario" (el 10 se escribe
  como 10a, 10b y 10c).
- [x] Cada escenario tiene al menos un requisito funcional que lo cubre.
- [x] Cada requisito es verificable y usa MUST/MUST NOT (FR-001 a FR-019).
- [x] Los criterios de éxito son medibles (SC-001 a SC-006).
- [x] Los casos límite están identificados (`sprint` inválido o ajeno, redondeos, todas sin Story Points, defecto
  detectado y resuelto en Sprints distintos, defecto de otro proyecto, Sprint sin registro de cierre, recarga).
- [x] Alcance y fuera de alcance definidos (HU-08, HU-09, HU-16, HU-25, histórico, porcentaje por Story Points).
- [x] Dependencias explícitas: HU-06, HU-07, HU-14, HU-15, HU-19, HU-20a, HU-20b; orden de implementación después de
  HU-06 y HU-15.
- [x] Los códigos 200/303/405/500/503 están definidos por caso.
- [x] Los textos de la pantalla se definen en un solo lugar.

## Preparación

- [x] La tabla de trazabilidad conecta escenario, requisito, éxito y prueba, con los mismos nombres de prueba que
  `tasks.md`.
- [x] Todos los requisitos y criterios de éxito tienen tareas en `tasks.md` (tabla "requisitos → tareas").
- [x] El plan respeta la constitución v1.1.0: puerto con adaptador PostgreSQL y pruebas de integración; RED →
  GREEN por bloque con `evidencia-red.txt`.
- [x] Sin migración nueva y sin editar migraciones existentes.
- [x] Los nombres de HU-06 y HU-15 que todavía no están en `main` quedan marcados para confirmar en T001.
- [ ] Aceptada por el equipo (completar al subir el Issue).

## Notas

- Las decisiones del equipo están registradas en "Clarifications" (sesión 2026-10-28).
- `/speckit-analyze` (28/10) no encontró problemas críticos. Se corrigieron:
  - A1, en FR-001: `GET` y `HEAD`, y 405 para cualquier otro método;
  - U1, en T016: el Sprint de la prueba de coherencia cumple la validación de cierre de HU-07;
  - C1, en T022: se cubre "No disponible" en la pantalla;
  - I1, en "Depende de": HU-19 figura solo para la demo;
  - I2, en Supuestos: HU-13 todavía no está en `main`.
- Revisión de datos de prueba (28/10): T011 detalla el Sprint de detección del defecto reabierto y el defecto de «Beta».
  Las historias se reparten entre el Sprint 1 (18 y 8) y el 2 (3 de 4 y una sin estimar), porque con 3, 5, 8 y 2 no
  hay tres "Hecho" que sumen 8. T022 usa los dos Sprints.
- Solo queda sin marcar "Aceptada por el equipo", que se completa al subir el Issue.
