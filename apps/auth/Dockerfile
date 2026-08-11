ARG GO_BUILD_IMAGE=golang:1.26.5-alpine3.23@sha256:622e56dbc11a8cfe87cafa2331e9a201877271cbff918af53d3be315f3da88cc
ARG GO_RUNTIME_IMAGE=alpine:3.23@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40

FROM ${GO_BUILD_IMAGE} AS go-build
ARG SOURCE_DATE_EPOCH=0
ENV CGO_ENABLED=0 \
    SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}
WORKDIR /src
COPY apps/auth/go.mod apps/auth/go.sum ./
RUN go mod download
COPY apps/auth/ ./
RUN mkdir -p /out \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/auth ./cmd/auth

FROM ${GO_RUNTIME_IMAGE} AS runtime
COPY --from=go-build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN addgroup -S -g 10001 avia-auth \
    && adduser -S -D -H -u 10001 -G avia-auth avia-auth \
    && mkdir -p /app \
    && chown 10001:10001 /app
WORKDIR /app
ENV TZ=UTC
COPY --from=go-build --chown=10001:10001 /out/auth /app/auth
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --retries=3 CMD wget --quiet --output-document=/dev/null http://127.0.0.1:8080/health/live || exit 1
ENTRYPOINT ["/app/auth"]
