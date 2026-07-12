FROM golang:1.26 AS deps

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

FROM deps AS build

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/backupbot ./cmd/backupbot
RUN CGO_ENABLED=0 GOOS=linux GOBIN=/out go install github.com/jackc/tern/v2@latest

FROM postgres:17-alpine AS runtime

RUN adduser -D -H -u 10001 appuser
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/backupbot /app/backupbot
COPY --from=build /out/tern /app/tern
COPY migrations /app/migrations

USER appuser
EXPOSE 8080
CMD ["/app/api"]
