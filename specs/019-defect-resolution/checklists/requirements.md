# Checklist de calidad de la especificación: HU-19

- [x] La historia sigue el formato "Como…, quiero…, para…".
- [x] No quedan "(a definir)" ni preguntas abiertas (decisión del equipo: bloquear sin Sprint activo).
- [x] Los 9 escenarios están en Dado/Cuando/Entonces, sin "Esquema del escenario".
- [x] Cada escenario tiene al menos un requisito funcional que lo cubre.
- [x] Cada requisito es verificable y usa MUST/MUST NOT.
- [x] Los criterios de éxito son medibles y no dependen de la tecnología.
- [x] Alcance y fuera de alcance definidos.
- [x] Dependencias explícitas: HU-06, HU-04/FIX-HU-04, HU-20a, HU-20b.
- [x] Los códigos 303/404/409/500 están definidos por caso.
- [x] Los mensajes están definidos en un solo lugar.
- [x] La tabla de trazabilidad conecta escenario, requisito, éxito y prueba.
- [x] El plan respeta la constitución v1.1.0 (puerto + adaptador PostgreSQL + integración).
- [x] No se crea migración nueva; el CHECK de HU-06 se respeta.
- [x] Los archivos de HU-06 que se editan están listados y coordinados (T004).
- [x] Los nombres tomados de `specs/006-defects` están marcados para verificar en T001.
- [x] Las consecuencias de reabrir sobre HU-10 están documentadas.
- [x] Estimación sugerida registrada: 3 SP (a confirmar en Planning Poker).
- [ ] Aceptada por el equipo (completar al subir el Issue).