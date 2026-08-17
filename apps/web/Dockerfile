ARG NODE_BUILD_IMAGE=node:24.16.0-alpine3.23@sha256:2bdb65ed1dab192432bc31c95f94155ca5ad7fc1392fb7eb7526ab682fa5bf14
ARG GO_BUILD_IMAGE=golang:1.26.5-alpine3.23@sha256:622e56dbc11a8cfe87cafa2331e9a201877271cbff918af53d3be315f3da88cc

FROM ${NODE_BUILD_IMAGE} AS web-build
WORKDIR /workspace/apps/web
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci --ignore-scripts --no-audit --no-fund
COPY docs/regulatory-sources/ /workspace/docs/regulatory-sources/
COPY apps/web/ ./
COPY assets/ /workspace/assets/

FROM web-build AS demo-build
ARG AVIA_APP_SHELL_PREDECESSOR_JSON=""
ARG AVIA_APP_SHELL_LEGACY_PREDECESSOR_JSON=""
ENV AVIA_APP_SHELL_PREDECESSOR_JSON=${AVIA_APP_SHELL_PREDECESSOR_JSON} \
    AVIA_APP_SHELL_LEGACY_PREDECESSOR_JSON=${AVIA_APP_SHELL_LEGACY_PREDECESSOR_JSON}
RUN npm run build:demo \
    && find dist/demo -type f -name '*.map' -delete \
    && node scripts/assert-app-shell-artifact.mjs dist/demo

FROM web-build AS http-build
ARG VITE_AVIA_DISABLE_BROWSER_TELEMETRY=0
ARG AVIA_APP_SHELL_PREDECESSOR_JSON=""
ARG AVIA_APP_SHELL_LEGACY_PREDECESSOR_JSON=""
ENV AVIA_APP_SHELL_PREDECESSOR_JSON=${AVIA_APP_SHELL_PREDECESSOR_JSON} \
    AVIA_APP_SHELL_LEGACY_PREDECESSOR_JSON=${AVIA_APP_SHELL_LEGACY_PREDECESSOR_JSON}
RUN VITE_AVIA_DISABLE_BROWSER_TELEMETRY="${VITE_AVIA_DISABLE_BROWSER_TELEMETRY}" npm run build:http \
    && find dist/http -type f -name '*.map' -delete \
    && node scripts/assert-app-shell-artifact.mjs dist/http \
    && ! grep -E '/src/(mock|entry/http-test)(/|\\.tsx?$)|seed-(data|visual-runtime)|test-?profile' dist/http/build-inputs.json

FROM ${GO_BUILD_IMAGE} AS web-server-build
WORKDIR /src
COPY deploy/local/webserver/main.go ./
RUN mkdir -p /out \
    && CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/web-server ./main.go

FROM scratch AS web-runtime
COPY --from=web-server-build /out/web-server /web-server
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/web-server"]

FROM web-runtime AS demo
ARG AVIA_SOURCE_COMMIT=unknown
ARG AVIA_TARGET=unknown
ARG AVIA_BUILD_PROFILE=demo
LABEL org.opencontainers.image.source="https://github.com/aviason/aviaSurveil" \
      org.opencontainers.image.revision="${AVIA_SOURCE_COMMIT}" \
      com.avia.target="${AVIA_TARGET}" \
      com.avia.build.profile="${AVIA_BUILD_PROFILE}" \
      com.avia.build.target="demo"
COPY --from=demo-build --chown=10001:10001 /workspace/apps/web/dist/demo/ /srv/

FROM web-runtime AS http
ARG AVIA_SOURCE_COMMIT=unknown
ARG AVIA_TARGET=unknown
ARG AVIA_BUILD_PROFILE=http
LABEL org.opencontainers.image.source="https://github.com/aviason/aviaSurveil" \
      org.opencontainers.image.revision="${AVIA_SOURCE_COMMIT}" \
      com.avia.target="${AVIA_TARGET}" \
      com.avia.build.profile="${AVIA_BUILD_PROFILE}" \
      com.avia.build.target="http"
COPY --from=http-build --chown=10001:10001 /workspace/apps/web/dist/http/ /srv/
