resource "google_access_context_manager_service_perimeter" "pass" {
  parent = "accessPolicies/1234567890"
  name   = "accessPolicies/1234567890/servicePerimeters/enforced"
  title  = "enforced-perimeter"

  status {
    restricted_services = ["storage.googleapis.com"]
  }
}
