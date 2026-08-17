resource "stackit_modelserving_token" "inference" {
  project_id   = "8e1e0b09-2f5a-4c0d-9f04-1b6b2a3c4d5e"
  name         = "inference"
  ttl_duration = "24h"
}
