resource "google_access_context_manager_service_perimeter" "fail" {
  parent = "accessPolicies/1234567890"
  name   = "accessPolicies/1234567890/servicePerimeters/dryrun"
  title  = "dryrun-perimeter"

  use_explicit_dry_run_spec = true

  spec {
    restricted_services = ["storage.googleapis.com"]
  }
}
