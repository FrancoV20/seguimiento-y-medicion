# Checklist de calidad de la especificación: HU-11

**Propósito**: validar que la especificación está completa antes de implementar.
**Creada**: 2026-10-27
**Feature**: [spec.md](../spec.md)

## Calidad del contenido

- [x] La historia sigue el formato "Como…, quiero…, para…".
- [x] Los escenarios no contienen detalles de implementación.
- [x] Enfocada en el valor para el usuario.
- [x] Todas las secciones obligatorias están completas (Sección SDD: intención, alcance, fuera de alcance,
  entradas, salidas, reglas, restricciones, condiciones de error).

## Completitud de requisitos

- [x] No quedan marcadores [NEEDS CLARIFICATION], "(a definir)" ni preguntas abiertas.
- [x] Los 16 escenarios del Issue están en Dado/Cuando/Entonces simples, sin "Esquema del escenario" (los que
  combinan casos con "o" se escriben como 2a/2b, 4a/4b, 6a/6b/6c, 12a/12b, 13a/13b y 14a/14b).
- [x] Cada escenario tiene al menos un requisito funcional que lo cubre.
- [x] Cada requisito es verificable y usa MUST/MUST NOT (FR-001 a FR-023).
- [x] Los criterios de éxito son medibles (SC-001 a SC-007).
- [x] Los casos límite están identificados (identificadores inválidos, proyecto borrado al guardar, orden de
  validación, varios integrantes con horas, bordes de fechas, concurrencia, escape, recarga).
- [x] Alcance y fuera de alcance definidos.
- [x] Dependencias explícitas: HU-01, FIX-HU-01, HU-03, HU-04, HU-20a, HU-20b, HU-21.
- [x] Los códigos 303/422/409/404/500 están definidos por caso.
- [x] Los mensajes nuevos se definen en un solo lugar, junto a los de HU-21.

## Preparación

- [x] La tabla de trazabilidad conecta escenario, requisito, éxito y prueba, con los mismos nombres de prueba que
  `tasks.md`.
- [x] Todos los requisitos y criterios de éxito tienen tareas en `tasks.md` (tabla "requisitos → tareas").
- [x] El plan respeta la constitución v1.1.0: puerto con adaptador PostgreSQL y pruebas de integración; RED →
  GREEN por bloque con `evidencia-red.txt`.
- [x] Sin migración nueva y sin editar migraciones existentes.
- [ ] Aceptada por el equipo (completar al subir el Issue).

## Notas

- Resuelto en `/speckit-clarify` el 27/10 (dos preguntas, una por vez): la carrera con la carga de horas se cierra
  quitando la pertenencia antes de contar horas y bloqueándola en modo compartido en el guardado de HU-03 (R11).
  Solo los Sprints que el cambio deja fuera bloquean el cambio de fechas (R6, escenario 11b nuevo).
- `/speckit-analyze` (27/10) no encontró problemas críticos. Se aplicaron U1 (FR-013 y R7 descritos como
  consecuencia de R6), U2 (el servicio recorta el nombre), I1 (este checklist) e I2 (mismo integrante en el
  escenario 15 y en T058). Quedan abiertos, de severidad baja: C1, C2, I3, A1, O1 y O2 (ver el resumen de analyze).
- Estimación: 5 SP (sugerida). Con 86 tareas y cambios en archivos de HU-21 y HU-03, conviene revisarla en el
  Planning Poker.
