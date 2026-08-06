# Non-compliant: Binary Authorization is present but explicitly disabled.
resource "google_container_cluster" "fail_disabled" {
  name     = "fail-disabled"
  location = "us-central1"

  binary_authorization {
    evaluation_mode = "DISABLED"
  }
}
