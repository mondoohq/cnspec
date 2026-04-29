# Cloud Memorystore Redis fixture for the tf-pass test bundle.
#
# Only google_redis_instance is declared - no google_redis_cluster - so the
# cluster-only checks (cluster-iam-auth-enabled, cluster-deletion-protection-
# enabled) do not run. The instance enables AUTH, transit encryption, CMEK,
# and uses the Standard HA tier.

resource "google_redis_instance" "cache" {
  name           = "cache-${random_id.rnd.hex}"
  tier           = "STANDARD_HA"
  memory_size_gb = 1
  region         = var.region

  auth_enabled           = true
  transit_encryption_mode = "SERVER_AUTHENTICATION"
  customer_managed_key   = google_kms_crypto_key.key.id
}
