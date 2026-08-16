resource "google_compute_forwarding_rule" "app" {
  name                  = "app-lb"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  all_ports             = false
  ports                 = ["8080", "8443"]
  ip_protocol           = "TCP"
}
