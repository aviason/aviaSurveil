ARG GO_BUILD_IMAGE=golang:1.26.5-alpine3.23@sha256:622e56dbc11a8cfe87cafa2331e9a201877271cbff918af53d3be315f3da88cc

FROM ${GO_BUILD_IMAGE} AS gateway-build
ARG CADDY_VERSION=v2.11.4
ARG GRPC_VERSION=v1.82.1
WORKDIR /src/caddy
RUN go mod download "github.com/caddyserver/caddy/v2@${CADDY_VERSION}" \
    && caddy_module="$(go env GOMODCACHE)/github.com/caddyserver/caddy/v2@${CADDY_VERSION}" \
    && cp -R "${caddy_module}/." /src/caddy/ \
    && go mod edit -require="google.golang.org/grpc@${GRPC_VERSION}" \
    && go mod tidy
RUN mkdir -p /out \
    && CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/caddy ./cmd/caddy
COPY deploy/local/gateway/healthcheck.go /src/healthcheck.go
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/healthcheck /src/healthcheck.go \
    && mkdir -p /runtime/data /runtime/config \
    && chown -R 1000:1000 /runtime

FROM scratch AS gateway
ENV XDG_CONFIG_HOME=/config
ENV XDG_DATA_HOME=/data
COPY --from=gateway-build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=gateway-build /out/caddy /caddy
COPY --from=gateway-build /out/healthcheck /healthcheck
COPY --from=gateway-build --chown=1000:1000 /runtime/data /data
COPY --from=gateway-build --chown=1000:1000 /runtime/config /config
COPY deploy/local/gateway/Caddyfile /etc/caddy/Caddyfile
COPY deploy/local/gateway/security-headers.caddy /etc/caddy/security-headers.caddy
COPY deploy/local/gateway/security-headers-http.caddy /etc/caddy/security-headers-http.caddy
USER 1000:1000
EXPOSE 8443
ENTRYPOINT ["/caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
