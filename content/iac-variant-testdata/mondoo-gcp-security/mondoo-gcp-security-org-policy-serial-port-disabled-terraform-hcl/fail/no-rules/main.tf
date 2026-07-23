# Non-compliant: constraint referenced but spec has no enforcing rules.
resource "google_org_policy_policy" "serial_port" {
  name   = "projects/my-project/policies/compute.disableSerialPortAccess"
  parent = "projects/my-project"

  spec {
    reset = true
  }
}
