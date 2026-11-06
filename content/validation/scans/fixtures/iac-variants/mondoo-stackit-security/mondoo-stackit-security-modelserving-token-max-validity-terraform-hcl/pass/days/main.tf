resource "stackit_modelserving_token" "inference" {
  project_id   = "00000000-0000-0000-0000-000000000000"
  name         = "inference-token"
  ttl_duration = "365d"
}
