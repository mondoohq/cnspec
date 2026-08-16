# Compliant: an org policy for compute.vmExternalIpAccess exists.
resource "google_org_policy_policy" "vm_external_ip" {
  name   = "projects/my-project/policies/compute.vmExternalIpAccess"
  parent = "projects/my-project"

  spec {
    rules {
      deny_all = "TRUE"
    }
  }
}
