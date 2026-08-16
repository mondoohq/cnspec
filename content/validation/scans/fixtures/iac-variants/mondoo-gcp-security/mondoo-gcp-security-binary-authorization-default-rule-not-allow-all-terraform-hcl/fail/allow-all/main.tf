# Non-compliant: default admission rule allows all images unconditionally.
resource "google_binary_authorization_policy" "fail_example" {
  default_admission_rule {
    evaluation_mode  = "ALWAYS_ALLOW"
    enforcement_mode = "ENFORCED_BLOCK_AND_AUDIT_LOG"
  }
}
