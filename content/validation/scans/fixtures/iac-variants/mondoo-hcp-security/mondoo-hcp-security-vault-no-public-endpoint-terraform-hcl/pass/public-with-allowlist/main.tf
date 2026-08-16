resource "hcp_vault_cluster" "secrets" {
  cluster_id      = "secrets"
  hvn_id          = hcp_hvn.main.hvn_id
  public_endpoint = true

  ip_allowlist {
    address     = "203.0.113.0/24"
    description = "corporate egress"
  }
}
