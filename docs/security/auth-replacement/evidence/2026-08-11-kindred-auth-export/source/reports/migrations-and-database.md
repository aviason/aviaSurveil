# Database and migrations inventory

The reviewed auth boundary uses DynamoDB user/app tables and optional DynamoDB Local tests. No PostgreSQL client, schema, migration directory, SQL file, cleanup job for PostgreSQL, or PostgreSQL integration test exists in the repository. This requirement is therefore `not applicable`, not a missing export artifact.

The user row carries identity, credential, verification, lockout, lifecycle, token-version and one refresh-session state. The app table carries consent/analytics records and unrelated product records. The copied storage and account-purge sources show the DynamoDB access, uniqueness locks and deletion path; unrelated product record types are documented in `source-reference/dependency-map.md`.
