# Compliant: org policy enforces compute.disableSerialPortAccess.
resource "google_org_policy_policy" "serial_port" {
  name   = "projects/my-project/policies/compute.disableSerialPortAccess"
  parent = "projects/my-project"

  spec {
    rules {
      enforce = "TRUE"
    }
  }
}
