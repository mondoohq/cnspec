# Compliant: Redis cluster encrypted with a customer-managed key.
resource "google_redis_cluster" "prod" {
  name        = "cmek-cluster"
  region      = "us-central1"
  shard_count = 3
  kms_key     = "projects/my-project/locations/us-central1/keyRings/redis/cryptoKeys/cmek"
}
