# Compliant: the rule admits only instances running as the named service account, not the internet.
resource "google_compute_firewall" "rule" {
  name    = "source-service-account"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["3389"]
  }

  source_service_accounts = ["jump@example-project.iam.gserviceaccount.com"]
}
