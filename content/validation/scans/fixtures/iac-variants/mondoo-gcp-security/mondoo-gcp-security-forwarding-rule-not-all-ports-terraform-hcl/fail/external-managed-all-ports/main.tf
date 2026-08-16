resource "google_compute_forwarding_rule" "wide_managed" {
  name                  = "wide-managed-lb"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  all_ports             = true
  ip_protocol           = "TCP"
}
