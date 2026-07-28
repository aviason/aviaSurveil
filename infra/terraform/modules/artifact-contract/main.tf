variable "image_uri" {
  description = "Private ECR runtime image pinned by SHA-256 digest."
  type        = string

  validation {
    condition     = can(regex("@sha256:[0-9a-f]{64}$", var.image_uri))
    error_message = "image_uri must be pinned by an immutable SHA-256 digest."
  }
}

variable "sbom_sha256" {
  description = "SHA-256 of the reviewed CycloneDX SBOM."
  type        = string

  validation {
    condition     = can(regex("^[0-9a-f]{64}$", var.sbom_sha256))
    error_message = "sbom_sha256 must be a lower-case SHA-256 digest."
  }
}

variable "scan_sha256" {
  description = "SHA-256 of the reviewed vulnerability scan result."
  type        = string

  validation {
    condition     = can(regex("^[0-9a-f]{64}$", var.scan_sha256))
    error_message = "scan_sha256 must be a lower-case SHA-256 digest."
  }
}

output "image_uri" {
  description = "Reviewed immutable image URI."
  value       = var.image_uri
}

output "sbom_sha256" {
  description = "Reviewed SBOM digest."
  value       = var.sbom_sha256
}

output "scan_sha256" {
  description = "Reviewed scan-result digest."
  value       = var.scan_sha256
}
