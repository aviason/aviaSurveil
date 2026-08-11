# Archive safety scan

This file records the final-tree and final-ZIP safety checks. A zero finding means the scanner found no matching artifact in the sanitized export; it is not a claim about the original repository or runtime secrets.

## Final tree scan

- Regular files: 112
- Symlinks: 0
- Forbidden paths/extensions (`.git`, `.env*`, private-key/container/database/archive artifacts): 0
- Private-key PEM markers: 0
- AWS access-key patterns: 0
- JWT-like three-segment token patterns: 0
- Production URL patterns (API Gateway/AppSync/Kindred/Avia domains): 0
- Allowlisted synthetic test placeholders (`fixture-secret`): 8 occurrences in sanitized test fixtures only; these are not credentials or runtime configuration.

## ZIP traversal scan

After packaging, `unzip -t` completed successfully; a Python `zipfile` traversal checked every entry and found no absolute path, `..` component, symlink entry, or duplicate filename. The ZIP contained the same 112 regular files and no forbidden paths/extensions. Private-key, AWS-key, JWT-like-token and production-URL scans over the decompressed ZIP contents returned zero matches; the eight synthetic fixture placeholders remained the only allowlisted secret-like strings.

The final ZIP SHA-256 is reported in the handoff response and is intentionally not duplicated here.
