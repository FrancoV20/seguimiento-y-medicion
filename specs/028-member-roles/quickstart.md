# Guía rápida: HU-28 – Roles del Proyecto (mínimos)

## Requisitos previos

- Go 1.27.1 (`go version`), Docker Desktop en ejecución y `golang-migrate/migrate` según el
  [README.md](../../README.md).
- **HU-27 integrada en la rama base** (T001: HU-27): sin ella no se empieza. Además, HU-11, HU-15, HU-05b y HU-18 para
  cubrir todas las rutas (T001).
- Rama: `git switch main; git pull; git switch -c HU-28`, solo si no hay cambios sin commitear.
- `DATABASE_URL` con el puerto de tu `docker-compose.yml` o `docker-compose.override.yml`:
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- Base al día con las migraciones de HU-27 y HU-28: `migrate -path db/migrations -database $env:DATABASE_URL up`. Sin
  migrar, el servidor no arranca (verificación de esquema de HU-20a).

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run "HU28|ScrumMaster|ParseRole|CanManage|PostgreSQLStore|ProjectMemberRole|ResolveContext|AssignRole" -v ./src/... ./db/... ./test/...
go test -count=1 -p 1 -cover ./src/role/... ./src/web/... ./src/cmd/...
gofmt -l src/role src/cmd/set-role test db
go vet ./...
```

`-count=1` evita la caché y `-p 1` evita que los paquetes de integración se pisen en la misma base.

## Escenarios de validación requeridos

Los valores esperados están en "Datos de ejemplo", "Rutas protegidas" y "Condiciones de error" de [spec.md](spec.md),
y en las tablas de [data-model.md](data-model.md).

1. **Dominio**: las tablas de `ParseRole` y `CanManage`.
2. **Almacenamiento** (PostgreSQL):
   - la migración sube (filas existentes en 'Integrante', el `CHECK` rechaza 'Product Owner'), baja y vuelve a subir;
   - `ResolveContext` devuelve el rol y `HasScrumMaster` de la tabla;
   - `role.PostgreSQLStore` asigna y responde los dos errores.
3. **Envoltorio**: los 5 pasos y el 2b, con un contexto armado a mano; `h` se llama solo en el paso 3.
4. **Rutas**: la tabla de rutas protegidas presentes, con Beto (403, "solo el Scrum Master") y con Dani en «Beta» (403,
   "no tiene Scrum Master"), sin llamar a la acción; votar, Tablero y esfuerzo sí llegan.
5. **Comando**: Beto pasa a Scrum Master de «Alfa»; rol, proyecto e integrante inválidos; opciones y entorno.
6. **BDD**: los 13 escenarios con el cableado de `main.go`.
7. **Regresión**: toda la suite en verde con las pruebas existentes identificándose como Scrum Master.

## Verificación manual

1. Migrar y levantar la aplicación (`go run ./src/cmd/server`).
2. Ver los números: `docker compose exec postgres psql -U sym_user -d seguimiento_y_medicion -c "SELECT p.id, p.name,
   m.id, m.display_name, pm.role FROM project_members pm JOIN projects p ON p.id = pm.project_id JOIN members m ON
   m.id = pm.member_id ORDER BY p.id, m.id"`. Todos figuran como "Integrante".
3. Armar «Alfa» (Ana, Beto, Caro) y «Beta» (Dani) con HU-21, con «Login» y «Perfil» en el Backlog de «Alfa» y
   «Reportes» en el de «Beta» (HU-22). Configurar las contraseñas con `set-password` (HU-27).
4. `go run ./src/cmd/set-role -proyecto <Alfa> -integrante <Ana> -rol "Scrum Master"` → "Ana ahora es Scrum Master de
   «Alfa»".
5. Identificarse como Ana en «Alfa» → "Integrante: Ana (Scrum Master)" en la estructura común. Como Ana, dejar el
   Sprint 2 en curso con «Login» (HU-23 y HU-15) y abrir la ronda 1 de «Login» (HU-05a). Antes del paso 4 no se podía,
   porque nadie era Scrum Master.
6. Identificarse como Beto → "Beto (Integrante)". Intentar:
   - cerrar el Sprint 2;
   - crear un Sprint;
   - agregar o quitar «Perfil»;
   - abrir, cerrar o empezar una ronda, o registrar la estimación;
   - guardar cambios en «Alfa».

   Cada intento muestra "Esta acción la puede hacer solo el Scrum Master del proyecto" y no cambia nada.
7. Como Beto, votar en «Login», mover una historia en el Tablero y registrar esfuerzo: funcionan.
8. "Salir" e intentar cerrar el Sprint 2 sin identificarse: se pide identificarse; nada cambia.
9. Como Ana con «Alfa» activo, abrir el formulario de modificar «Beta» y guardar: 403 "Para modificar este proyecto,
   elegilo como proyecto activo e identificate como su Scrum Master"; «Beta» no cambia (escenario 13).
10. Elegir «Beta», identificarse como Dani e intentar crear un Sprint: "Este proyecto no tiene Scrum Master asignado.
    Pedí que se configure con el comando de roles".
11. `set-role` con `-rol "Product Owner"`, con Dani en «Alfa» y con un proyecto 999999: "El rol no es válido", "El
    integrante no pertenece al proyecto" y "El proyecto no existe"; la consulta del paso 2 no cambia.
12. `set-role` para Beto como "Scrum Master" en «Alfa» y, sin volver a identificarse, cerrar el Sprint 2 como Beto:
    se cierra como indica HU-15.
13. Detener la base (`docker compose stop`) e intentar una acción de Scrum Master: página 503 sin detalles.

## Evidencias a guardar en el PR

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 -Append specs/028-member-roles/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/028-member-roles/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/028-member-roles/cobertura.txt
```

`evidencia-red.txt` se completa por secciones (Dominio y migración, Contexto con rol, Envoltorio y rutas, Comando y
Pruebas existentes). Cada sección se guarda antes de escribir el código de su bloque.
