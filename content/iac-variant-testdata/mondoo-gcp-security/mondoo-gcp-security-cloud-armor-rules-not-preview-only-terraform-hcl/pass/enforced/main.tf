# Compliant: all rules are enforced (preview not enabled).
resource "google_compute_security_policy" "pass_example" {
  name = "pass-policy"

  rule {
    action   = "deny(403)"
    priority = 1000
    preview  = false
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["192.0.2.0/24"]
      }
    }
    description = "Block bad range"
  }

  rule {
    action   = "allow"
    priority = 2147483647
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    description = "Default rule"
  }
}
