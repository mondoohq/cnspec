# With roles omitted the credential inherits the creating user's roles minus
# any that are explicitly restricted, which is the common least-privilege form.
resource "openstack_identity_application_credential_v3" "backup_agent" {
  name        = "backup-agent"
  description = "Credential used by the nightly backup agent"
  expires_at  = "2027-01-01T00:00:00Z"
}
