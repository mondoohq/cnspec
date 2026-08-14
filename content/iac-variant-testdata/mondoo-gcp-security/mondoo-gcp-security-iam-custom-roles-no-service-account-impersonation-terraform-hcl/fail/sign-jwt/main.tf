# signJwt mints assertions a service account can exchange for access tokens,
# which is impersonation by another name.
resource "google_project_iam_custom_role" "token_minter" {
  project     = var.project_id
  role_id     = "tokenMinter"
  title       = "Token Minter"
  description = "Used by the workload identity broker"
  stage       = "GA"

  permissions = [
    "iam.serviceAccounts.get",
    "iam.serviceAccounts.signJwt",
  ]
}
