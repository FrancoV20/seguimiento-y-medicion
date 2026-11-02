# Despliegue en Azure

La aplicación se publica en **Azure Container Apps** y usa una base **Azure Database for
PostgreSQL – Flexible Server**. Todo vive en el grupo de recursos `frsrutn` de la suscripción *Azure for Students* de la cátedra,
en la región `switzerlandnorth`, que es donde está el entorno de Container Apps que creó la cátedra.

| Recurso | Nombre |
|---|---|
| Servidor PostgreSQL | `grupo7-seguimiento-medicion` |
| Base de datos | `seguimiento_y_medicion` |
| Entorno de Container Apps | `utnapp-env` (de la cátedra, compartido) |
| Aplicación web | `grupo7-seguimiento-medicion-app` |
| Registro de imágenes | `cac8d295feabacr` |

Dirección pública: https://grupo7-seguimiento-medicion-app.braveocean-c591f3f5.switzerlandnorth.azurecontainerapps.io/

> El grupo `frsrutn` es compartido con otros equipos: todos nuestros recursos empiezan con
> `grupo7-` y no se tocan los que no son nuestros.

## Requisitos

- Azure CLI con sesión iniciada con las claves de la cátedra (service principal) y los valores
  por defecto configurados:
  ```powershell
  az account show -o table
  az config set defaults.group=frsrutn defaults.location=southcentralus
  ```
- `golang-migrate` instalado (ver README).
- `Dockerfile` y `.dockerignore` en la raíz del repositorio.
- Docker Desktop (solo para usar `psql` del contenedor local en el paso 4) y Go (comandos de administración).

Todos los comandos son para PowerShell y se ejecutan desde la raíz del repositorio.

## 1. Variables

```powershell
$RG       = "frsrutn"
$LOC      = "switzerlandnorth"
$DB_SERVER = "grupo7-seguimiento-medicion"
$DB_NAME  = "seguimiento_y_medicion"
$DB_USER  = "symadmin"
$APP      = "grupo7-seguimiento-medicion-app"
$ENV_NAME = "utnapp-env"
```

Claves (solo letras y números, para no tener que codificarlas en la URL):

```powershell
$chars = [char[]]((48..57) + (65..90) + (97..122))
$DB_PASS = "Sym" + (-join (1..24 | ForEach-Object { $chars | Get-Random })) + "9"
$COOKIE_SECRET = -join (1..48 | ForEach-Object { $chars | Get-Random })
$DB_PASS
```

Guardar `$DB_PASS` en un lugar seguro (gestor de contraseñas). **Nunca** en el repositorio.

## 2. Extensiones y proveedores

```powershell
az extension add --name containerapp --upgrade
foreach ($p in "Microsoft.App", "Microsoft.OperationalInsights", "Microsoft.ContainerRegistry", "Microsoft.DBforPostgreSQL") {
  "$p -> " + (az provider show --namespace $p --query registrationState -o tsv)
}
```

Todos tienen que decir `Registered`. Si alguno dice `NotRegistered`, probar
`az provider register --namespace <nombre>`; si da error de permisos, pedírselo a la cátedra.

## 3. Base de datos

Crear el servidor (tarda entre 5 y 10 minutos). `--public-access 0.0.0.0` permite conexiones desde
servicios de Azure, como la aplicación:

```powershell
az postgres flexible-server create --resource-group $RG --name $DB_SERVER --location $LOC `
  --admin-user $DB_USER --admin-password $DB_PASS `
  --tier Burstable --sku-name Standard_B1ms --storage-size 32 --version 16 `
  --public-access 0.0.0.0 --yes
```

Crear la base y habilitar la IP de la PC desde donde se corren las migraciones:

```powershell
az postgres flexible-server db create --resource-group $RG --server-name $DB_SERVER --name $DB_NAME
$MY_IP = Invoke-RestMethod https://api.ipify.org
az postgres flexible-server firewall-rule create --resource-group $RG --server-name $DB_SERVER `
  --name pc-$env:USERNAME --start-ip-address $MY_IP --end-ip-address $MY_IP
```

## 4. Migraciones

```powershell
$AZ_DB_URL = "postgres://${DB_USER}:${DB_PASS}@${DB_SERVER}.postgres.database.azure.com:5432/${DB_NAME}?sslmode=require"
migrate -path db/migrations -database $AZ_DB_URL up
migrate -path db/migrations -database $AZ_DB_URL version
```

`version` tiene que devolver el número de la última migración de `db/migrations`.

Cargar los integrantes del equipo, usando el `psql` del contenedor local:

```powershell
docker compose up -d postgres
Get-Content db/seeds/members.sql | docker exec -i seguimiento-y-medicion-db psql $AZ_DB_URL
docker exec -i seguimiento-y-medicion-db psql $AZ_DB_URL -c "SELECT id, display_name FROM members ORDER BY id;"
```

Los `id` no siguen el orden del archivo: usar los que muestra la última consulta. Configurar la
contraseña de cada integrante (la pide dos veces, de 8 a 128 caracteres):

```powershell
$env:DATABASE_URL = $AZ_DB_URL
go run ./src/cmd/set-password -integrante <id>
```

## 5. Publicar la aplicación

> `utnapp-env` es un entorno *express*: no admite identidades administradas y el service principal
> no puede asignar roles. Por eso la app entra al registro de imágenes con usuario y contraseña, y
> **no** se usa `az containerapp up` (configura el registro con una identidad y falla).

Registro de imágenes del grupo (ya creado). Si no existiera, crearlo con
`az acr create --resource-group $RG --name <nombre> --sku Basic --location $LOC`.

```powershell
$ACR = "cac8d295feabacr"
az acr update --name $ACR --admin-enabled true
$ACR_PASS = az acr credential show --name $ACR --query "passwords[0].value" -o tsv
```

Compilar la imagen en Azure con el `Dockerfile` (no hace falta Docker local):

```powershell
$IMAGE = "$ACR.azurecr.io/${APP}:v1"
az acr build --registry $ACR --image "${APP}:v1" .
```

Crear la aplicación con el registro, las claves y las variables de entorno:

```powershell
az containerapp create --name $APP --resource-group $RG --environment $ENV_NAME --image $IMAGE `
  --registry-server "$ACR.azurecr.io" --registry-username $ACR --registry-password $ACR_PASS `
  --ingress external --target-port 8080 `
  --secrets "database-url=$AZ_DB_URL" "cookie-secret=$COOKIE_SECRET" `
  --env-vars "DATABASE_URL=secretref:database-url" "COOKIE_SECRET=secretref:cookie-secret" `
  --min-replicas 0 --max-replicas 1
```

## 6. Verificar

```powershell
az containerapp show --name $APP --resource-group $RG --query properties.configuration.ingress.fqdn -o tsv
```

Abrir `https://<esa dirección>` en el navegador. Si no responde, mirar los logs:

```powershell
az containerapp logs show --name $APP --resource-group $RG --tail 50
```

## 7. Asignar el Scrum Master

El rol necesita un proyecto: crearlo primero desde la web (**Proyectos**) y después buscar su `id`:

```powershell
docker exec -i seguimiento-y-medicion-db psql $AZ_DB_URL -c "SELECT id, name FROM projects ORDER BY id;"
$env:DATABASE_URL = $AZ_DB_URL
go run ./src/cmd/set-role -proyecto <id del proyecto> -integrante <id> -rol "Scrum Master"
```

## Actualizar después de cambios

1. Si hay migraciones nuevas, aplicarlas primero (paso 4, solo `migrate ... up`).
2. Desde `main` actualizado, compilar una imagen con una etiqueta nueva (`v2`, `v3`, …) y
   apuntar la app a ella:
   ```powershell
   az acr build --registry $ACR --image "${APP}:v2" .
   az containerapp update --name $APP --resource-group $RG --image "$ACR.azurecr.io/${APP}:v2"
   ```

## Cuidar el crédito

- La aplicación escala a cero cuando no recibe visitas (`--min-replicas 0`); la primera visita
  tarda unos segundos más.
- La base cobra por hora aunque no se use. Apagarla cuando no se trabaja:
  ```powershell
  az postgres flexible-server stop  --resource-group $RG --name $DB_SERVER
  az postgres flexible-server start --resource-group $RG --name $DB_SERVER
  ```
  Azure la vuelve a encender sola a los 7 días.

## Problemas comunes

| Mensaje | Qué hacer |
|---|---|
| `location is restricted` al crear la base | Repetir el paso 3 con otra región permitida (`$LOC = "eastus2"`, por ejemplo) |
| `MissingSubscriptionRegistration` | Falta registrar un proveedor (paso 2); pedirlo a la cátedra |
| `connection refused` / `timeout` en `migrate` | La IP de la PC cambió: repetir la regla de firewall con la IP nueva |
| `password authentication failed` | `$DB_PASS` no es la del servidor (se perdió al cerrar PowerShell) |
| La app muestra error de esquema | Faltan migraciones: paso 4 |
| `ExpressEnvironmentFeatureNotSupported` (identidad) | La app se creó con `az containerapp up`: borrarla y crearla como en el paso 5 |
| `roleAssignments/write` al publicar | Mismo caso: no usar `up`, usar el paso 5 |
