# Non-compliant: a port range of 1-65535 forwards every port without setting all_ports.
resource "google_compute_forwarding_rule" "nlb" {
  name                  = "nlb"
  load_balancing_scheme = "EXTERNAL"
  port_range            = "1-65535"
  ip_protocol           = "TCP"
  backend_service       = google_compute_region_backend_service.default.id
}
