# Compliant: DENY rule for the public internet.
resource "google_app_engine_firewall_rule" "pass_example" {
  action       = "DENY"
  source_range = "*"
  priority     = 100
}
