# Non-compliant: global policy evaluation is explicitly disabled.
resource "google_binary_authorization_policy" "fail_example" {
  global_policy_evaluation_mode = "DISABLE"

  default_admission_rule {
    evaluation_mode  = "REQUIRE_ATTESTATION"
    enforcement_mode = "ENFORCED_BLOCK_AND_AUDIT_LOG"

    require_attestations_by = [
      "projects/my-project/attestors/prod-attestor",
    ]
  }
}
