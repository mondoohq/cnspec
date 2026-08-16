# Compliant: ALLOW rule scoped to a specific CIDR, not the public internet.
resource "google_app_engine_firewall_rule" "pass_example" {
  action       = "ALLOW"
  source_range = "10.0.0.0/8"
  priority     = 100
}
