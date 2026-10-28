# Guía rápida: HU-05b – Revelación de Votos y Detección de Divergencia

## Requisitos previos

- Go 1.27.1 (`go version`), Docker Desktop en ejecución y `golang-migrate/migrate` según el
  [README.md](../../README.md).
- HU-05a, HU-20a y HU-20b en `main` (ya integradas).
- Rama: `git switch main; git pull; git switch -c HU-05b`.
- `DATABASE_URL` con el puerto de tu `docker-compose.yml` o `docker-compose.override.yml`:
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- Base con las migraciones hasta `011` aplicadas (`migrate -path db/migrations -database $env:DATABASE_URL up`).
  HU-05b no agrega migraciones.

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run "HU05b|Spread|CheckClose|CloseRound|PokerCatalog" -v ./src/poker/... ./src/web/... ./test/...
go test -count=1 -p 1 -cover ./src/poker/...
```

Como en el CI: `gofmt -l .` (sin salida), `go vet ./...` y `go test ./... -p 1 -race`.

## Escenarios de validación requeridos

1. **Regla** (sin base): la tabla de `TestSpread` de [data-model.md](data-model.md) (2-13-5 diverge, 3-5-8 no, 3-13-3
   sí, 5-5 no, un voto no, 1-3 no, 1-5 sí, 55-89 no, 1-89 sí), vacío → `ErrNoVotes`, valor 4 →
   `ErrValueNotInSeries`.
2. **Cierre** (PostgreSQL): «Login» se cierra con `closed_at`; «Perfil» se cierra con dos votos; «Ayuda» → `ErrNoVotes`
   y sigue abierta; cerrar dos veces → `ErrRoundClosed` con la misma `closed_at`; inexistente y de «Beta» →
   `ErrRoundNotFound`; un voto que espera al cierre → `ErrRoundClosed` y Caro sigue en 5; dos cierres simultáneos →
   uno OK y otro `ErrRoundClosed`; un cierre con el contexto vencido no deja cambios.
3. **Catálogo** (PostgreSQL): rondas cerradas de la más reciente a la más antigua; valores y "No votó"; integrante
   quitado con su voto; ninguna ronda abierta ni valor de ronda abierta en `ClosedRounds`.
4. **Pantalla**: los 12 escenarios con sus códigos (303/409/404/500), recarga sin reenviar, textos escapados, ronda
   cerrada sin votos sin fallar y "Aún no hay rondas cerradas".
5. **Seguridad**: con «Login» cerrada y «Pagos» abierta, la página de Beto no cambia ni un byte cuando Ana cambia su
   voto de «Pagos».

## Verificación manual

1. Preparar «Alfa» con Ana, Beto y Caro y las historias «Login», «Pagos», «Reportes», «Perfil» y «Ayuda» (HU-21,
   HU-22). Identificarse como Ana y abrir una ronda para cada historia en `http://localhost:8080/planning-poker`.
2. Votar en ventanas privadas distintas (una por integrante) los valores de [spec.md](spec.md), "Datos de ejemplo":
   «Login» 2/13/5, «Pagos» 3/5/8, «Reportes» 3/13/3, «Perfil» 5/5 (Caro no vota), «Ayuda» sin votos.
3. Antes de cerrar: "Aún no hay rondas cerradas" y ningún valor ajeno visible.
4. Como Ana, "Cerrar ronda" en «Login»: "Ronda cerrada para la historia «Login»"; en "Rondas cerradas", Ana 2, Beto 13,
   Caro 5 y la alerta "Advertencia: Divergencia: los votos van de 2 a 13. Debatan antes de acordar la estimación".
5. Como Beto, recargar: los mismos valores y la misma alerta (escenario 7).
6. Cerrar «Pagos»: "Sin divergencia: los votos van de 3 a 8", sin alerta; queda primera en "Rondas cerradas".
7. Cerrar «Reportes»: alerta "de 3 a 13". Cerrar «Perfil»: Caro "No votó" y "Sin divergencia: los votos van de 5 a 5".
8. "Cerrar ronda" en «Ayuda»: "La ronda no tiene votos" y sigue abierta.
9. Como Caro, desde una pestaña abierta antes del cierre, votar 8 en «Login»: "La ronda ya está cerrada"; su voto
   sigue 5. Reenviar el cierre de «Login» desde esa pestaña: "La ronda ya está cerrada".
10. Cambiar a mano el campo `ronda` del formulario (herramientas del navegador) por un id inexistente: "La ronda no
    existe".
11. Salir de la identificación y enviar un cierre: se pide identificarse y nada se cierra.

## Evidencias a guardar en el PR

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 -Append specs/005b-poker-reveal/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/005b-poker-reveal/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/005b-poker-reveal/cobertura.txt
```

`evidencia-red.txt` se completa por secciones (Divergencia, Cierre, Pantalla), cada una antes del código de su bloque.
