# Granting admin to an application credential hands full project control to
# whatever holds the secret.
resource "openstack_identity_application_credential_v3" "ci_deploy" {
  name        = "ci-deploy"
  description = "Credential used by the deployment pipeline"
  roles       = ["admin", "member"]
  expires_at  = "2027-01-01T00:00:00Z"
}
