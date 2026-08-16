resource "openstack_identity_application_credential_v3" "monitoring" {
  name        = "monitoring"
  description = "Application credential for the monitoring service"
  expires_at  = ""

  roles = ["reader"]
}
