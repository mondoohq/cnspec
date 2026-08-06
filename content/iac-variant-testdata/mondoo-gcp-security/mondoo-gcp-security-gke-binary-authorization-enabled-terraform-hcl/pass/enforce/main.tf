# Compliant: the cluster enforces the project Binary Authorization policy.
resource "google_container_cluster" "pass_enforce" {
  name     = "pass-enforce"
  location = "us-central1"

  binary_authorization {
    evaluation_mode = "PROJECT_SINGLETON_POLICY_ENFORCE"
  }
}
