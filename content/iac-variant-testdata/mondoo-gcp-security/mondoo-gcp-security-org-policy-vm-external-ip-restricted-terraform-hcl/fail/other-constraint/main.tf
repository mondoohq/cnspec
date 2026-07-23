# Non-compliant: org policies exist, but none restrict compute.vmExternalIpAccess.
resource "google_org_policy_policy" "shielded_vm" {
  name   = "projects/my-project/policies/compute.requireShieldedVm"
  parent = "projects/my-project"

  spec {
    rules {
      enforce = "TRUE"
    }
  }
}
