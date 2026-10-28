# Guía rápida: HU-26 – Cookies firmadas

## Requisitos previos

- HU-20b en `main` (ya está).
- PostgreSQL de pruebas disponible (el mismo que usan las demás historias).
- Rama: `git checkout -b feature/hu-26-cookies-firmadas`

## Configurar la clave

PowerShell (solo para esa ventana):

```powershell
$env:COOKIE_SECRET = "una-clave-larga-de-al-menos-32-caracteres-123"
```

Sin la variable, la aplicación arranca con clave temporal y deja un aviso en el
registro.

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run HU26 ./test/...
go test -count=1 -p 1 -cover ./src/web/... ./src/cmd/...
```

## Verificación manual

1. Arrancar la aplicación con `go run ./src/cmd/server`, elegir un proyecto e
   identificarse.
2. En las herramientas del navegador (Aplicación → Cookies) comprobar que
   `ctx_proyecto` y `ctx_integrante` tienen la forma `número.firma`.
3. Cambiar el número de `ctx_proyecto` a mano y recargar: el sistema vuelve a
   pedir el proyecto y las cookies desaparecen.
4. Elegir proyecto otra vez, dejar `ctx_integrante` sin firma (solo el número) y
   entrar a una acción que requiere integrante: pide identificarse.
5. Reiniciar con la misma `COOKIE_SECRET`: el contexto se mantiene.
6. Reiniciar con otra `COOKIE_SECRET`: se vuelve a pedir el proyecto.
7. Probar con una clave de menos de 32 caracteres: la aplicación no arranca y
   explica el largo mínimo.

## Evidencias a guardar en el PR

- `specs/026-signed-cookies/evidencia-red.txt`
- `specs/026-signed-cookies/evidencia-pruebas.txt`
- `specs/026-signed-cookies/cobertura.txt`