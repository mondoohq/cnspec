resource "openstack_identity_application_credential_v3" "deploy" {
  name         = "deploy"
  expires_at   = "2026-12-31T23:59:59Z"
  unrestricted = true

  roles = ["member"]
}
