FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /dnsmasq-leases-ui .
FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=build /dnsmasq-leases-ui /app/dnsmasq-leases-ui
COPY templates /app/templates
COPY static /app/static
USER app
EXPOSE 5000
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s CMD wget -qO- http://127.0.0.1:${PORT:-5000}/leases >/dev/null || exit 1
ENTRYPOINT ["/app/dnsmasq-leases-ui"]
