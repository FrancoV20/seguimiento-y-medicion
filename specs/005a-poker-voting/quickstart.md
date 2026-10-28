# Guía rápida: HU-05a – Emisión de voto oculto en Planning Poker

## Requisitos previos

- Go 1.27.1 (`go version`), Docker Desktop en ejecución y `golang-migrate/migrate` según el
  [README.md](../../README.md).
- HU-01, HU-02, HU-20a y HU-20b en `main` (ya integradas).
- Rama: `git switch main; git pull; git switch -c HU-05a`. Si `HU-05a` ya existe localmente (hoy apunta a la
  `main` vieja, sin commits propios: confirmarlo con `git log main..HU-05a`), usar
  `git switch HU-05a; git merge --ff-only origin/main`.
- `DATABASE_URL` con el puerto de tu `docker-compose.yml` o `docker-compose.override.yml`:
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.

## Migración

```powershell
docker compose up -d postgres
migrate -path db/migrations -database $env:DATABASE_URL up
migrate -path db/migrations -database $env:DATABASE_URL down 1
migrate -path db/migrations -database $env:DATABASE_URL up
```

Esperado: `up` crea `poker_rounds` y `poker_votes`; `down 1` las borra; el segundo `up` las vuelve a crear. Los
CHECK rechazan un estado distinto de 'Abierta'/'Cerrada', una ronda 'Cerrada' sin `closed_at` y un voto con
valor 4. Esquema en [data-model.md](data-model.md).

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run "HU05a|Poker|Series|ParseValue|CheckOpen|CheckVote|Service|Postgres|Migracion" -v ./src/poker/... ./src/web/... ./test/...
go test -count=1 -p 1 -cover ./src/poker/...
```

`-count=1` evita la caché y `-p 1` evita que los paquetes de integración se pisen en la misma base.

## Escenarios de validación requeridos

1. **Dominio** (sin base): `ParseValue` con cada valor de la serie, vacío, 0, 4, 90, "abc", "05", " 5", "5.0";
   `TestSeries_CoincideConStoryPoints`; `CheckOpen` sin rondas, con una abierta y con una cerrada; `CheckVote`
   abierta y cerrada; `Service` con identificadores en 0 y con un almacén que falla.
2. **Adaptador** (PostgreSQL): abrir OK, historia inexistente, de otro proyecto, con ronda abierta, con ronda
   cerrada, concurrente; votar nuevo, reemplazo, ronda cerrada, inexistente, de otro proyecto, falla sin
   cambios; abrir no modifica la historia.
3. **Catálogo** (PostgreSQL): todas las historias del proyecto; solo rondas abiertas del proyecto; todos los
   integrantes con `HasVoted`; `OwnVote` solo del integrante pedido; proyecto vacío.
4. **Handlers**: los 13 escenarios de la spec con sus códigos (303/422/409/404/500), recarga sin reenviar, título
   con `<script>` escapado y falla de lectura.
5. **Ocultamiento**: con Ana en 13 y Beto en 3, la fila de Ana en la página de Beto no tiene "13"; al cambiar Ana a
   21, la página de Beto no cambia ni un byte.

## Verificación manual

1. Elegir un proyecto con al menos dos integrantes y dos historias (HU-21, HU-22) e identificarse como el
   primero (`/contexto/integrante`).
2. Abrir `http://localhost:8080/planning-poker`: se ve el selector de historias y "No hay rondas abiertas".
3. Abrir una ronda para la primera historia: "Ronda abierta para la historia «…»" y la ronda 1 con todos los
   integrantes "Sin votar".
4. Votar 5: "Voto registrado", tu fila dice "Votó" y "Tu voto: 5". Votar 8: "Voto actualizado".
5. Identificarse como el segundo integrante (otra ventana privada) y votar 13: en la ventana del primero, al
   recargar, el segundo aparece "Votó" sin valor.
6. Intentar abrir otra ronda para la misma historia: "Ya hay una ronda abierta para esta historia".
7. En SQL, cerrar la ronda (`UPDATE poker_rounds SET status = 'Cerrada', closed_at = now() WHERE id = …`) y votar
   desde la pantalla abierta antes: "La ronda ya está cerrada". Abrir otra ronda para esa historia: "La historia
   ya tuvo una ronda de votación".
8. Salir de la identificación y entrar a Planning Poker: se pide identificarse y se vuelve a la pantalla.
9. Con un proyecto sin historias: "Aún no hay historias" y "No hay rondas abiertas", sin formulario de apertura.

## Evidencias a guardar en el PR

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 -Append specs/005a-poker-voting/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/005a-poker-voting/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/005a-poker-voting/cobertura.txt
```

`evidencia-red.txt` se completa por secciones (Fundamentos, Abrir ronda, Votar, Errores y estados, Pantalla),
cada una antes del código de su bloque.
