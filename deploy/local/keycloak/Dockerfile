ARG KEYCLOAK_IMAGE=quay.io/keycloak/keycloak:26.7.0@sha256:0f198be292568439d700cdbfb893e69a6009bb43a94a06a945b1d3d506c76b13
ARG NODE_BUILD_IMAGE=node:24.16.0-alpine3.23@sha256:2bdb65ed1dab192432bc31c95f94155ca5ad7fc1392fb7eb7526ab682fa5bf14

FROM ${KEYCLOAK_IMAGE} AS keycloak-source

FROM ${NODE_BUILD_IMAGE} AS keycloak-patches

WORKDIR /workspace

COPY deploy/local/keycloak/runtime-patches.json /workspace/runtime-patches.json
COPY deploy/local/keycloak/download-runtime-patches.mjs /workspace/download-runtime-patches.mjs
COPY deploy/local/keycloak/rewrite-quarkus-classpath.mjs /workspace/rewrite-quarkus-classpath.mjs
COPY --from=keycloak-source /opt/keycloak/lib/lib/deployment/appmodel.dat /metadata/appmodel.dat
COPY --from=keycloak-source /opt/keycloak/lib/lib/deployment/deployment-class-path.dat /metadata/deployment-class-path.dat
COPY --from=keycloak-source /opt/keycloak/lib/quarkus/quarkus-application.dat /metadata/quarkus-application.dat

RUN node /workspace/download-runtime-patches.mjs \
    /workspace/runtime-patches.json \
    /patches \
    && node /workspace/rewrite-quarkus-classpath.mjs \
    /workspace/runtime-patches.json \
    /metadata/appmodel.dat \
    /metadata/deployment-class-path.dat \
    /metadata/quarkus-application.dat

FROM ${KEYCLOAK_IMAGE} AS keycloak-build

ENV KC_DB=postgres \
    KC_HEALTH_ENABLED=true \
    KC_HTTP_RELATIVE_PATH=/identity

RUN rm -f \
    /opt/keycloak/bin/client/keycloak-admin-cli-26.7.0.jar \
    /opt/keycloak/lib/lib/main/com.microsoft.sqlserver.mssql-jdbc-13.2.1.jre11.jar \
    /opt/keycloak/lib/lib/main/org.postgresql.postgresql-42.7.11.jar \
    /opt/keycloak/lib/lib/main/com.fasterxml.jackson.core.jackson-core-2.21.2.jar \
    /opt/keycloak/lib/lib/main/com.fasterxml.jackson.core.jackson-databind-2.21.2.jar \
    /opt/keycloak/lib/lib/main/io.micrometer.micrometer-*-1.16.3.jar \
    /opt/keycloak/lib/lib/main/io.netty.netty-*-4.1.135.Final*.jar

COPY --from=keycloak-patches --chown=1000:0 /patches/ /opt/keycloak/lib/lib/main/
COPY --from=keycloak-patches --chown=1000:0 /metadata/appmodel.dat /opt/keycloak/lib/lib/deployment/appmodel.dat
COPY --from=keycloak-patches --chown=1000:0 /metadata/deployment-class-path.dat /opt/keycloak/lib/lib/deployment/deployment-class-path.dat
COPY --from=keycloak-patches --chown=1000:0 /metadata/quarkus-application.dat /opt/keycloak/lib/quarkus/quarkus-application.dat

RUN /opt/keycloak/bin/kc.sh build

FROM ${KEYCLOAK_IMAGE} AS keycloak

COPY --from=keycloak-build /opt/keycloak/ /opt/keycloak/

RUN rm -f \
    /opt/keycloak/bin/client/keycloak-admin-cli-26.7.0.jar \
    /opt/keycloak/lib/lib/main/com.microsoft.sqlserver.mssql-jdbc-13.2.1.jre11.jar \
    /opt/keycloak/lib/lib/main/org.postgresql.postgresql-42.7.11.jar \
    /opt/keycloak/lib/lib/main/com.fasterxml.jackson.core.jackson-core-2.21.2.jar \
    /opt/keycloak/lib/lib/main/com.fasterxml.jackson.core.jackson-databind-2.21.2.jar \
    /opt/keycloak/lib/lib/main/io.micrometer.micrometer-*-1.16.3.jar \
    /opt/keycloak/lib/lib/main/io.netty.netty-*-4.1.135.Final*.jar
