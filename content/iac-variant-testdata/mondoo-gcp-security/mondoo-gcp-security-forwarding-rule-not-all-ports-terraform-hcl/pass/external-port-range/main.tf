resource "google_compute_forwarding_rule" "https" {
  name                  = "https-lb"
  load_balancing_scheme = "EXTERNAL"
  port_range            = "443"
  ip_protocol           = "TCP"
  target                = google_compute_target_https_proxy.default.id
}
