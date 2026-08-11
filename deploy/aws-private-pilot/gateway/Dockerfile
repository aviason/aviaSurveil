ARG NODE_BUILD_IMAGE=node:24.16.0-alpine3.23@sha256:2bdb65ed1dab192432bc31c95f94155ca5ad7fc1392fb7eb7526ab682fa5bf14
ARG GO_BUILD_IMAGE=golang:1.26.5-alpine3.23@sha256:622e56dbc11a8cfe87cafa2331e9a201877271cbff918af53d3be315f3da88cc

FROM ${NODE_BUILD_IMAGE} AS web-build
WORKDIR /workspace/apps/web
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci --ignore-scripts --no-audit --no-fund
COPY docs/regulatory-sources/ /workspace/docs/regulatory-sources/
COPY apps/web/ ./
COPY assets/ /workspace/assets/
RUN npm run build:http \
    && find dist/http -type f -name '*.map' -delete \
    && node scripts/assert-app-shell-artifact.mjs dist/http \
    && ! grep -E '/src/(mock|entry/http-test)(/|\\.tsx?$)|seed-(data|visual-runtime)|test-?profile' dist/http/build-inputs.json

FROM ${GO_BUILD_IMAGE} AS gateway-build
ARG CADDY_VERSION=v2.11.4
ARG GRPC_VERSION=v1.82.1
ARG X_TEXT_VERSION=v0.39.0
WORKDIR /src/caddy
RUN go mod download "github.com/caddyserver/caddy/v2@${CADDY_VERSION}" \
    && caddy_module="$(go env GOMODCACHE)/github.com/caddyserver/caddy/v2@${CADDY_VERSION}" \
    && cp -R "${caddy_module}/." /src/caddy/ \
    && go mod edit -require="google.golang.org/grpc@${GRPC_VERSION}" \
    && go mod edit -require="golang.org/x/text@${X_TEXT_VERSION}" \
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
COPY --from=web-build --chown=1000:1000 /workspace/apps/web/dist/http/ /srv/
COPY deploy/aws-private-pilot/gateway/Caddyfile /etc/caddy/Caddyfile
USER 1000:1000
EXPOSE 8080
ENTRYPOINT ["/caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
