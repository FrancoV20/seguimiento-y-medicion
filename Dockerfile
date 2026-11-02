# Imagen de la aplicación web para Azure Container Apps.
# Etapa 1: compila el servidor (los templates, el CSS y las migraciones van embebidos en el binario).
FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./src/cmd/server

# Etapa 2: imagen final liviana, sin el código fuente ni Go.
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app
USER app
COPY --from=build /out/server /usr/local/bin/server
ENV ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["server"]
