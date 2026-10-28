# Checklist de calidad de la especificación: HU-13

**Propósito**: validar que la especificación está completa antes de implementar.
**Creada**: 2026-10-27
**Feature**: [spec.md](../spec.md)

## Calidad del contenido

- [x] La historia sigue el formato "Como…, quiero…, para…".
- [x] Los escenarios no contienen detalles de implementación.
- [x] Enfocada en el valor para el Scrum Master.
- [x] Todas las secciones obligatorias están completas (Sección SDD: intención, alcance, fuera de alcance,
  entradas, salidas, reglas, restricciones, condiciones de error).

## Completitud de requisitos

- [x] No quedan marcadores [NEEDS CLARIFICATION], "(a definir)" ni preguntas abiertas.
- [x] Los 15 escenarios del Issue están en Dado/Cuando/Entonces simples, sin "Esquema del escenario" (los que
  combinan casos con "o" se escriben como 2a/2b, 7a–7d, 8a/8b, 10a–10c, 11a/11b, 12a/12b y 13a–13c).
- [x] Cada escenario tiene al menos un requisito funcional que lo cubre.
- [x] Cada requisito es verificable y usa MUST/MUST NOT (FR-001 a FR-024).
- [x] Los criterios de éxito son medibles (SC-001 a SC-006).
- [x] Los casos límite están identificados (identificadores inválidos, proyecto activo cambiado, filtro con
  mayúsculas o repetido, orden al cambiar la prioridad, criterios vacíos, cierre simultáneo del Sprint, guardados
  simultáneos, escape, recarga).
- [x] Alcance y fuera de alcance definidos.
- [x] Dependencias explícitas: HU-02, HU-04, HU-20a, HU-20b, HU-22; se implementa después de HU-15 y HU-17.
- [x] Los códigos 200/303/422/409/404/500 están definidos por caso.
- [x] Los mensajes nuevos se definen en un solo lugar, junto a los de HU-22.

## Preparación

- [ ] La tabla de trazabilidad conecta escenario, requisito, éxito y prueba, con los mismos nombres de prueba que
  `tasks.md`.
- [ ] Todos los requisitos y criterios de éxito tienen tareas en `tasks.md` (tabla "requisitos → tareas").
- [ ] El plan respeta la constitución v1.1.0: puerto con adaptador PostgreSQL y pruebas de integración; RED →
  GREEN por bloque con `evidencia-red.txt`.
- [x] Sin migración nueva y sin editar migraciones existentes.
- [ ] Aceptada por el equipo (completar al subir el Issue).

## Notas

- Resuelto en `/speckit-clarify` el 27/10 (dos preguntas, una por vez): abrir la edición de una historia de un
  Sprint finalizado vuelve a la lista con una advertencia (R11, escenario 12c nuevo), y la edición conserva el filtro
  de la lista (R12, FR-024, escenario 7e nuevo).
- Los ítems de "Preparación" se completan después de `/speckit-plan`, `/speckit-tasks` y `/speckit-analyze`.
- `/speckit-analyze` (28/10): se aplicaron I1 (el `POST` lee el formulario antes del control de Sprint finalizado,
  para que el 409 muestre lo enviado), U1 (la edición no revisa el estado guardado del proyecto; research, decisión
  17) e I3 (orden de implementación después de HU-15 y HU-17 en la spec, el plan y el quickstart).
