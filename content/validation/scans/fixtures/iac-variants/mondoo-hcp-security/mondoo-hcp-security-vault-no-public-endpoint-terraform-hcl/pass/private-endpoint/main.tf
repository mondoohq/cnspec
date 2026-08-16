# public_endpoint defaults to false, so omitting it keeps the cluster off the internet.
resource "hcp_vault_cluster" "secrets" {
  cluster_id = "secrets"
  hvn_id     = hcp_hvn.main.hvn_id
}
