# Non-compliant: ALLOW rule open to the entire public internet.
resource "google_app_engine_firewall_rule" "fail_example" {
  action       = "ALLOW"
  source_range = "*"
  priority     = 100
}
