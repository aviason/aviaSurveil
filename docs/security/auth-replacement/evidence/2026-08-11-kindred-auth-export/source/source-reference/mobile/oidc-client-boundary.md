# Mobile identity boundary reference

The copied mobile files (`AuthService.swift`, the GraphQL clients, `AppSettingsStore.swift`, and auth tests) implement email/password registration/login, custom refresh/logout mutations, bearer headers and Keychain storage. They do not implement an OAuth/OIDC browser redirect, authorization code, PKCE, state, nonce, ID-token validation, UserInfo, or discovery client.

The original app container wires an AppSync issuer/audience for a GraphQL consumer and also contains operational URLs/API-key-like values. Those sections are not copied. This sanitized note is sufficient to classify the mobile path as a resource-server/bearer client rather than an OIDC relying party.
