# Checklist de calidad de la especificación: HU-27

**Propósito**: validar que la especificación está completa antes de implementar.
**Creada**: 2026-10-28
**Feature**: [spec.md](../spec.md)

## Calidad del contenido

- [x] La historia sigue el formato "Como…, quiero…, para…".
- [x] Los escenarios no contienen detalles de implementación.
- [x] Enfocada en el valor para el Integrante (nadie actúa a su nombre).
- [x] Todas las secciones obligatorias están completas (Sección SDD: intención, alcance, fuera de alcance,
  entradas, salidas, reglas, restricciones, condiciones de error).

## Completitud de requisitos

- [x] No quedan marcadores [NEEDS CLARIFICATION], "(a definir)" ni preguntas abiertas.
- [x] Los 12 escenarios del Issue están en Dado/Cuando/Entonces simples, con el texto del Issue.
- [x] Los datos de ejemplo son los del Issue; lo agregado para las pruebas (contraseña de Dani, Beto con la misma
  contraseña que Ana solo en el escenario 9) está marcado como agregado.
- [x] Las cuentas del bloqueo cierran (5 fallos → bloqueo de 60 s desde el quinto; 429 dentro; acierto a los 60 s).
- [x] Cada escenario tiene al menos un requisito funcional que lo cubre.
- [x] Cada requisito es verificable y usa MUST/MUST NOT (FR-001 a FR-021).
- [x] Los criterios de éxito son medibles (SC-001 a SC-006).
- [x] Los casos límite están identificados (identificación previa, espacios y Unicode, largo al identificarse, valor
  guardado mal formado, concurrencia, reinicio, bloqueo como molestia, cambio de contraseña con sesión abierta, salir
  sin identificación, comando sin opción, sin base, con entrada cerrada, con contraseña ya configurada).
- [x] Alcance y fuera de alcance definidos (roles, cambio y recuperación por web, vencimiento, invalidar sesiones,
  ocultar lo que se escribe en el comando).
- [x] Dependencias explícitas: HU-20b y HU-26; integración al final; archivos de otras historias que se tocan.
- [x] Los códigos 303/401/403/405/422/429/500/503 están definidos por caso, con qué cuenta como fallo.
- [x] Los textos de la pantalla y del comando se definen en un solo lugar.

## Preparación

- [x] La tabla de trazabilidad conecta escenario, requisito, éxito y prueba, con los mismos nombres de prueba que
  `tasks.md`.
- [x] Todos los requisitos y criterios de éxito tienen tareas en `tasks.md` (tabla "requisitos → tareas"), con 40
  tareas como máximo.
- [x] El plan respeta la constitución v1.1.0: puertos con adaptador PostgreSQL y pruebas de integración; RED →
  GREEN por bloque con `evidencia-red.txt`.
- [x] La migración nueva tiene up y down, con prueba, y su número se confirma en T001 y antes del PR.
- [ ] Aceptada por el equipo (completar al subir el Issue).

## Notas

- Las decisiones del equipo están registradas en "Clarifications" (sesión 2026-10-28).
- `/speckit-clarify` (28/10): sin preguntas. Las decisiones del equipo cubren todas las categorías; lo que quedaba
  abierto se resolvió en "Reglas", "Casos límite" y "Supuestos y decisiones" de spec.md:
  - el orden de las comprobaciones y qué cuenta como fallo;
  - el contador vuelve a cero al vencer el bloqueo;
  - un rechazo no toca la identificación previa;
  - la contraseña de Dani y Beto con la misma contraseña que Ana, solo en el escenario 9.
- `/speckit-analyze` (28/10): sin problemas críticos ni violaciones de la constitución. Cobertura de FR y SC del 100%
  (21 FR, 6 SC, 39 tareas). Se corrigieron:
  - I1, en la trazabilidad de spec.md: faltaba `TestSetPassword_ErrorAlGuardar`, que estaba en T019;
  - I2, en research.md (decisión 16): los ayudantes de `test/` reciben la clave, como dicen T024 y T031;
  - I3, en los riesgos de plan.md: los ayudantes se crean en T023 y T024, no en T030;
  - U1, en T023: la contraseña de más de 128 caracteres al identificarse da 401 y cuenta (caso límite sin prueba);
  - U2, en T023: "Salir" sin estar identificado (caso límite sin prueba);
  - U3, en T019: la `DATABASE_URL` inalcanzable tiene mensaje y código.

  Quedan documentados:
  - `<NNN>` (número de la migración): marcador intencional que fija T001 y reconfirma T038;
  - SC-004 (menos de 1 s): solo verificación manual (T037), porque con `-race` el tiempo varía;
  - escenario 10: la BDD guarda con el mismo puerto que el comando, que tiene su propia prueba de integración
    (`TestRun_ConfiguraContrasena`);
  - escenario 7: "registrar esfuerzo o votar" se prueba con una sección de prueba con `RequiresMember`, igual que en
    HU-20b y HU-26.
- Hallazgo de la verificación de nombres: las pruebas que se identifican por `/contexto/integrante` son 8, no 6 (nota
  6 del equipo); faltaban `src/web/effort_handlers_test.go` y `src/web/poker_view_test.go`. Además, hay 2 dobles de
  `ContextStore` en las pruebas de HU-23 que hay que completar (T016).
- Solo queda sin marcar "Aceptada por el equipo", que se completa al subir el Issue.
