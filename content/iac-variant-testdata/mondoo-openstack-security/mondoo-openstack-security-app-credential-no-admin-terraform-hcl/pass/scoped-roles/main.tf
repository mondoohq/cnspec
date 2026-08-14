resource "openstack_identity_application_credential_v3" "ci_deploy" {
  name         = "ci-deploy"
  description  = "Credential used by the deployment pipeline"
  roles        = ["member", "reader"]
  expires_at   = "2027-01-01T00:00:00Z"
  unrestricted = false
}
