# --- build stage ---
FROM golang:1.24-alpine AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 — статический бинарник, чтобы он работал в distroless без libc
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/catalog-service .

# --- runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/catalog-service /catalog-service

ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/catalog-service", "-healthcheck"]

ENTRYPOINT ["/catalog-service"]
