# Compliant: default admission rule requires attestation, not ALWAYS_ALLOW.
resource "google_binary_authorization_policy" "pass_example" {
  default_admission_rule {
    evaluation_mode  = "REQUIRE_ATTESTATION"
    enforcement_mode = "ENFORCED_BLOCK_AND_AUDIT_LOG"

    require_attestations_by = [
      "projects/my-project/attestors/prod-attestor",
    ]
  }
}
