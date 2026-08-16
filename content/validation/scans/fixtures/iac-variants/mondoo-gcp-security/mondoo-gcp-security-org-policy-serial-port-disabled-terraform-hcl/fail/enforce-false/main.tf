# Non-compliant: the serial port constraint is explicitly not enforced.
resource "google_org_policy_policy" "serial_port" {
  name   = "projects/my-project/policies/compute.disableSerialPortAccess"
  parent = "projects/my-project"

  spec {
    rules {
      enforce = "FALSE"
    }
  }
}
