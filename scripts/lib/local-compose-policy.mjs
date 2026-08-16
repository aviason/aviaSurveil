import path from "node:path";

const digestPattern = /@sha256:[a-f0-9]{64}$/u;
const secretKeyPattern =
  /(?:password|passwd|secret|token|credential|private[_-]?key)/iu;

function violation(code, service, message) {
  return { code, service, message };
}

function serviceSecretSources(service) {
  return new Set(
    (service.secrets ?? []).map((secret) =>
      typeof secret === "string" ? secret : secret.source,
    ),
  );
}

function serviceNetworkNames(service) {
  if (Array.isArray(service.networks)) {
    return service.networks;
  }
  return Object.keys(service.networks ?? {});
}

function volumeSourceAt(service, target) {
  for (const volume of service.volumes ?? []) {
    if (typeof volume === "string") {
      const [source, mountedAt] = volume.split(":");
      if (mountedAt === target) {
        return source;
      }
      continue;
    }
    if (volume.target === target) {
      return volume.source;
    }
  }
  return undefined;
}

function normalizedPort(port) {
  if (typeof port === "number" || typeof port === "string") {
    const fields = String(port).split(":");
    return {
      hostIp: fields.length === 3 ? fields[0] : "",
      published: Number(fields.at(-2) ?? fields[0]),
      target: Number(fields.at(-1)),
    };
  }
  return {
    hostIp: port.host_ip ?? "",
    published: Number(port.published),
    target: Number(port.target),
  };
}

function allowedPorts(policy, serviceName) {
  return [
    ...(policy.browserPublishedPorts?.[serviceName] ?? []),
    ...(policy.developerPublishedPorts?.[serviceName] ?? []),
  ];
}

function hasReviewedReason(exceptions, serviceName) {
  return (
    typeof exceptions?.[serviceName] === "string" &&
    exceptions[serviceName].trim().length >= 24
  );
}

function environmentEntries(environment) {
  if (Array.isArray(environment)) {
    return environment.map((entry) => {
      const separator = entry.indexOf("=");
      return separator === -1
        ? [entry, ""]
        : [entry.slice(0, separator), entry.slice(separator + 1)];
    });
  }
  return Object.entries(environment ?? {});
}

function isSecretPath(key, value) {
  if (key === "AVIA_ROSTER_CREDENTIAL_DIRECTORY") {
    return value === "/run/roster-credentials";
  }
  return (
    key.endsWith("_FILE") &&
    typeof value === "string" &&
    path.posix.isAbsolute(value) &&
    value.startsWith("/run/secrets/")
  );
}

export function validateComposePolicy({ compose, lock, policy, profile }) {
  const violations = [];
  const services = compose.services ?? {};
  const lockedReferences = new Set(
    Object.values(lock.images ?? {}).map((entry) => entry.reference),
  );

  if (profile) {
    const requiredServices = policy.profileServices?.[profile] ?? [];
    const allowedServices = new Set(requiredServices);
    for (const requiredService of requiredServices) {
      if (!services[requiredService]) {
        violations.push(
          violation(
            "MISSING_PROFILE_SERVICE",
            requiredService,
            `service is required by the ${profile} profile`,
          ),
        );
      }
    }
    for (const serviceName of Object.keys(services)) {
      if (!allowedServices.has(serviceName)) {
        violations.push(
          violation(
            "UNEXPECTED_PROFILE_SERVICE",
            serviceName,
            `service is not approved for the ${profile} profile`,
          ),
        );
      }
    }
  }

  for (const [serviceName, service] of Object.entries(services)) {
    for (const [key, value] of environmentEntries(service.environment)) {
      if (secretKeyPattern.test(key) && !isSecretPath(key, value)) {
        violations.push(
          violation(
            "PLAINTEXT_SECRET",
            serviceName,
            `${key} must reference an absolute /run/secrets file`,
          ),
        );
      }
    }

    const mountedSecrets = serviceSecretSources(service);
    for (const requiredSecret of policy.secretMounts?.[serviceName] ?? []) {
      if (!mountedSecrets.has(requiredSecret)) {
        violations.push(
          violation(
            "MISSING_SECRET_MOUNT",
            serviceName,
            `required secret ${requiredSecret} is not mounted`,
          ),
        );
      }
    }

    if (policy.externalImageServices?.includes(serviceName)) {
      if (!digestPattern.test(service.image ?? "")) {
        violations.push(
          violation(
            "UNPINNED_EXTERNAL_IMAGE",
            serviceName,
            "external image is not pinned by sha256 digest",
          ),
        );
      } else if (!lockedReferences.has(service.image)) {
        violations.push(
          violation(
            "IMAGE_LOCK_MISMATCH",
            serviceName,
            "external image reference is absent from image-lock.json",
          ),
        );
      }
    }

    const approvedPorts = allowedPorts(policy, serviceName);
    for (const rawPort of service.ports ?? []) {
      const port = normalizedPort(rawPort);
      const approved = approvedPorts.some(
        (candidate) =>
          candidate.hostIp === port.hostIp &&
          Number(candidate.published) === port.published &&
          Number(candidate.target) === port.target,
      );
      if (!approved) {
        violations.push(
          violation(
            "PUBLISHED_INTERNAL_PORT",
            serviceName,
            `published port ${port.hostIp}:${port.published}:${port.target} is not approved`,
          ),
        );
      }
    }

    if (!policy.publicNetworkServices?.includes(serviceName)) {
      const reviewedEgressNetworks = new Set(
        policy.egressNetworkServices?.[serviceName] ?? [],
      );
      for (const networkName of serviceNetworkNames(service)) {
        if (
          compose.networks?.[networkName]?.internal !== true &&
          !reviewedEgressNetworks.has(networkName)
        ) {
          violations.push(
            violation(
              "UNRESTRICTED_NETWORK",
              serviceName,
              `network ${networkName} is not internal`,
            ),
          );
        }
      }
    }

    const runtimeUser = String(service.user ?? "");
    if (runtimeUser.trim() === "") {
      violations.push(
        violation(
          "ROOT_USER",
          serviceName,
          "runtime user must be explicit so image defaults cannot silently change",
        ),
      );
    } else if (
      /^(?:0|root)(?::|$)/u.test(runtimeUser) &&
      !hasReviewedReason(policy.rootUserExceptions, serviceName)
    ) {
      violations.push(
        violation(
          "ROOT_USER",
          serviceName,
          "root runtime user lacks a reviewed policy reason",
        ),
      );
    }

    if (
      service.read_only !== true &&
      !hasReviewedReason(policy.writableRootfsExceptions, serviceName)
    ) {
      violations.push(
        violation(
          "WRITABLE_ROOTFS_EXCEPTION",
          serviceName,
          "writable root filesystem lacks a reviewed policy reason",
        ),
      );
    }

    if (
      policy.healthcheckRequiredServices?.includes(serviceName) &&
      !service.healthcheck?.test
    ) {
      violations.push(
        violation(
          "MISSING_HEALTHCHECK",
          serviceName,
          "required health check is absent",
        ),
      );
    }
  }

  const databasePolicy = policy.databaseIsolation;
  if (
    databasePolicy &&
    services[databasePolicy.applicationService] &&
    services[databasePolicy.identityService]
  ) {
    const applicationVolume = volumeSourceAt(
      services[databasePolicy.applicationService],
      databasePolicy.dataTarget,
    );
    const identityVolume = volumeSourceAt(
      services[databasePolicy.identityService],
      databasePolicy.dataTarget,
    );
    if (
      !applicationVolume ||
      !identityVolume ||
      applicationVolume === identityVolume
    ) {
      violations.push(
        violation(
          "SHARED_DATABASE",
          `${databasePolicy.applicationService},${databasePolicy.identityService}`,
          "application and identity databases must use distinct named volumes",
        ),
      );
    }
  }

  return violations;
}
