resource "google_compute_forwarding_rule" "wide" {
  name                  = "wide-lb"
  load_balancing_scheme = "EXTERNAL"
  all_ports             = true
  ip_protocol           = "TCP"
  backend_service       = google_compute_region_backend_service.default.id
}
