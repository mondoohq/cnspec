# Non-compliant: automatic acceptance accepts every consumer, so the accept list
# is never consulted.
resource "google_compute_service_attachment" "psc" {
  name                  = "psc-service"
  region                = "us-central1"
  enable_proxy_protocol = false
  connection_preference = "ACCEPT_AUTOMATIC"
  nat_subnets           = [google_compute_subnetwork.psc.id]
  target_service        = google_compute_forwarding_rule.producer.id

  consumer_accept_lists {
    project_id_or_num = "trusted-consumer-project"
    connection_limit  = 5
  }
}
