# Non-compliant: a rule is in preview mode, so it is not enforced.
resource "google_compute_security_policy" "fail_example" {
  name = "fail-policy"

  rule {
    action   = "deny(403)"
    priority = 1000
    preview  = true
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["192.0.2.0/24"]
      }
    }
    description = "Block bad range (preview only)"
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
