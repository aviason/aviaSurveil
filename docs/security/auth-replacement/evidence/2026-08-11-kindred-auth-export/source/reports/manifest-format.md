# Export manifest format

`manifest.json` lists each exported regular file and its SHA-256 digest, excluding the two manifest files themselves. `manifest.sha256` is the handoff checksum list for every other exported file, including `manifest.json`; it excludes only its own bytes to avoid a self-referential hash. The ZIP SHA-256 is calculated separately after archive creation and is supplied in the final handoff fields.
