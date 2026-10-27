# Guía rápida: HU-19 – Resolver y reabrir defectos

## Requisitos previos

- HU-06 mergeada en `main`.
- PostgreSQL de pruebas disponible (el mismo que usan las demás historias).
- Rama: `git checkout -b feature/hu-19-resolucion-defectos`

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run HU19 ./test/... ./src/web/...
go test -count=1 -p 1 -cover ./src/defect/... ./src/web/...
```

## Verificación manual

1. Elegir un proyecto con un Sprint activo y al menos un defecto registrado
   (HU-06).
2. Abrir `/defectos`: el defecto "Abierto" muestra **Resolver**.
3. Pulsar **Resolver**: el defecto pasa a "Resuelto", muestra "Sprint N" como Sprint
   de resolución y aparece el mensaje de éxito una sola vez.
4. Pulsar **Reabrir**: el defecto vuelve a "Abierto" y el Sprint de resolución
   vuelve a mostrar "—".
5. Cerrar el Sprint (HU-15) e intentar resolver otro defecto "Abierto": el sistema
   lo bloquea con el mensaje de falta de Sprint activo.

## Evidencias a guardar en el PR

- `specs/019-defect-resolution/evidencia-red.txt`
- `specs/019-defect-resolution/evidencia-pruebas.txt`
- `specs/019-defect-resolution/cobertura.txt`