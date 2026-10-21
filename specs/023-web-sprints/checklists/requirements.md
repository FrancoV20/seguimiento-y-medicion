## Notes

- Revisión del 21/10/2026: la spec pasó de 10 a 17 escenarios simples (sin esquemas), uno por regla.
  Se agregaron "Sin Sprint activo", "Agregar historias sin marcar ninguna" y los tres de quitar; se
  separaron Goal vacío, fechas y formato de fecha.
- Decisiones del equipo del 20/10/2026 incorporadas: un solo Sprint activo por proyecto (regla de
  HU-04, FIX-HU-04; la pantalla solo la informa) y cuatro estados de historia.
- "Quitar" ya no es opcional ni historia aparte: forma parte de HU-23 con su regla en el dominio
  (solo "En Sprint" y sin horas), porque `RemoveStory` de HU-04 no la valida (T047).
- Depende también de FIX-HU-04 y HU-03 (horas registradas). No depende de HU-22: las pruebas crean sus propias historias.
- Se sacaron del Issue "Sin proyecto activo al guardar", "Recarga" y "Error al leer" (HU-20b y HU-20a);
  siguen con prueba propia (T024, T013 y T032).
- Los textos propios de HU-23 son una propuesta del equipo; los de Goal, fechas e historia no
  disponible son los de HU-04; los de fechas faltantes y formato son los mismos de HU-21.
- Se cierran acá los pendientes de usabilidad de HU-04 (SC-009 y SC-010). Los umbrales de 90 % y 3
  minutos vienen de HU-04; confirmarlos antes de la Sprint Review.
- Pendiente de verificación (no bloquea la spec): firmas, errores y mensajes reales de HU-04 y
  FIX-HU-04, y la columna de historia en `effort_logs` (T001 y T004).
- Estimación votada en el Planning Poker: 5 Story Points sobre la tarjeta original. Agregar y quitar
  historias no estaban en esa tarjeta: conviene revisarla.
- No incluido: validar el Sprint contra las fechas del proyecto y congelar "SP planificados" al empezar
  el Sprint (decisiones del equipo).

All checklist items pass. The feature is ready for `/speckit-plan`.