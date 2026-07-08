resource "google_container_cluster" "primary" {
  name     = "primary"
  location = "us-central1"

  master_auth {
    username = "admin"
    password = "s3cr3t-p4ssw0rd-value"
  }
}
