resource "hcp_vault_cluster" "secrets" {
  cluster_id      = "secrets"
  hvn_id          = hcp_hvn.main.hvn_id
  public_endpoint = true
}
