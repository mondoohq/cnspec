# Non-compliant: the default rule (priority 2147483647) allows all traffic.
resource "google_compute_security_policy" "fail_example" {
  name = "fail-policy"

  rule {
    action   = "allow"
    priority = 2147483647
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    description = "Default allow rule"
  }
}
