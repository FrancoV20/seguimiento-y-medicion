# Guía rápida: HU-27 – Identificación con Contraseña

## Requisitos previos

- Go 1.27.1 (`go version`), Docker Desktop en ejecución y `golang-migrate/migrate` según el
  [README.md](../../README.md).
- HU-20b y HU-26 en `main` (ya están). HU-27 se integra **al final**, después de las demás historias.
- Rama: `git switch main; git pull; git switch -c HU-27`, solo si no hay cambios sin commitear.
- `DATABASE_URL` con el puerto de tu `docker-compose.yml` o `docker-compose.override.yml`:
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- Base al día con la migración nueva: `migrate -path db/migrations -database $env:DATABASE_URL up`. Sin migrar, el
  servidor no arranca (verificación de esquema de HU-20a).

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run "HU27|MemberPassword|MemberCredential|PasswordStore|Lockout|HashPassword|VerifyPassword|ValidateNewPassword|SetPassword|PasswordHash" -v ./src/... ./db/... ./test/...
go test -count=1 -p 1 -run "IteracionesReales" -v ./src/auth/...
go test -count=1 -p 1 -cover ./src/auth/... ./src/web/... ./src/cmd/...
gofmt -l src/auth src/cmd/set-password test db
go vet ./...
```

`-count=1` evita la caché y `-p 1` evita que los paquetes de integración se pisen en la misma base. `gofmt -l` no
tiene que listar ningún archivo nuevo.

## Escenarios de validación requeridos

Los valores esperados están en "Datos de ejemplo" y "Cuentas del bloqueo" de [spec.md](spec.md), y en las tablas de
[data-model.md](data-model.md).

1. **Dominio** (sin base):
   - dos valores guardados de "ana-clave-2026" son distintos y los dos verifican;
   - "otra-clave" no verifica;
   - un valor de 1.000 iteraciones verifica;
   - los valores mal formados de la tabla dan `ErrMalformedHash`;
   - la tabla de `ValidateNewPassword`;
   - el bloqueo: 5 fallos → bloqueado; a los 59 s, sigue; a los 60 s, libre y con el contador en cero.
2. **Almacenamiento** (PostgreSQL):
   - `MemberCredential` de Ana (con valor), Caro (sin valor), Dani en «Alfa» (no pertenece) y base cerrada
     (`Unavailable`);
   - `PostgreSQLPasswordStore` guarda y responde `ErrMemberNotFound`;
   - la migración sube, baja y vuelve a subir.
3. **Comando**: Caro con "caro-clave-2026" dos veces → "Contraseña configurada para Caro" y la contraseña verifica; las
   contraseñas cortas, largas o distintas no cambian nada; integrante inexistente; uso incorrecto; sin
   `DATABASE_URL`.
4. **Pantalla**:
   - los 12 escenarios, con reloj simulado para el 5;
   - los códigos 303, 401, 422, 429, 500 y 503;
   - `GET /contexto/salir` responde 405;
   - la contraseña no aparece en la base, en las páginas ni en el registro.
5. **Regresión**: toda la suite en verde con las 8 pruebas existentes actualizadas.

## Verificación manual

1. Migrar la base y levantar la aplicación (`go run ./src/cmd/server`).
2. Ver los números de los integrantes: `docker compose exec postgres psql -U sym_user -d seguimiento_y_medicion -c
   "SELECT id, display_name FROM members ORDER BY id"`.
3. Con HU-21, crear «Alfa» con Ana, Beto y Caro y «Beta» con Dani. Si no existen, cargarlos antes con
   `db/seeds/members.sql` o con SQL.
4. Configurar contraseñas:
   - `go run ./src/cmd/set-password -integrante <id de Ana>` y escribir dos veces "ana-clave-2026" → "Contraseña
     configurada para Ana";
   - lo mismo para Beto ("beto-clave-2026") y Dani ("dani-clave-2026").
5. Probar el comando con Caro: "corta" dos veces → "La contraseña debe tener al menos 8 caracteres";
   "caro-clave-2026" y "caro-clave-2027" → "Las contraseñas no coinciden". En los dos casos, `password_hash` de Caro
   sigue en `NULL`.
6. Elegir «Alfa» y entrar a "Registro de esfuerzo": se pide identificarse. Elegir a Ana con "ana-clave-2026" → "Te
   identificaste como Ana", su nombre en la estructura común y vuelta a "Registro de esfuerzo". Medir a ojo que tarda
   menos de 1 segundo (SC-004).
7. Presionar "Salir" → "Cerraste tu identificación"; el nombre desaparece y el proyecto sigue elegido.
8. Ana con "otra-clave" → "Integrante o contraseña incorrectos", Ana marcada y el campo vacío. Beto sin contraseña →
   "Ingresá tu contraseña" junto al campo. Caro con cualquier contraseña → "Caro todavía no tiene contraseña
   configurada. Pedile al Scrum Master que la configure".
9. Ana con "otra-clave" 5 veces y después con "ana-clave-2026" → "Demasiados intentos fallidos. Esperá un minuto y
   volvé a intentar". Esperar un minuto y repetir → identificada.
10. Configurar Caro con "caro-clave-2026" e identificarse con ella (escenario 10).
11. Identificada como Ana, cambiar a «Beta» y volver a «Alfa»: no hay integrante y hay que ingresar la contraseña.
12. Revisar `SELECT display_name, password_hash FROM members`: solo valores `pbkdf2-sha256$600000$…`, ninguno
    igual a otro. Revisar la consola del servidor: no aparece ninguna contraseña.
13. Detener la base (`docker compose stop`) e intentar identificarse: página 503 sin detalles, sin identificar.

## Evidencias a guardar en el PR

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 -Append specs/027-member-password/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/027-member-password/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/027-member-password/cobertura.txt
```

`evidencia-red.txt` se completa por secciones (Dominio, Almacenamiento, Comando, Pantalla y Pruebas existentes).
Cada sección se guarda antes de escribir el código de su bloque; en "Pruebas existentes", antes de actualizarlas.
