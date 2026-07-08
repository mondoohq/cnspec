# Compliant: manual approval required for consumer connections.
resource "google_compute_service_attachment" "psc" {
  name                  = "psc-service"
  region                = "us-central1"
  enable_proxy_protocol = false
  connection_preference = "ACCEPT_MANUAL"
  nat_subnets           = [google_compute_subnetwork.psc.id]
  target_service        = google_compute_forwarding_rule.producer.id
}
