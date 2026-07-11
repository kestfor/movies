FROM golang:1.26 AS deps

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

FROM deps AS build

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.22 AS runtime

RUN adduser -D -H -u 10001 appuser
WORKDIR /app
COPY --from=build /out/api /app/api

USER appuser
EXPOSE 8080
CMD ["/app/api"]
