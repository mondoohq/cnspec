resource "hcp_vault_cluster" "secrets" {
  cluster_id = "secrets"
  hvn_id     = hcp_hvn.main.hvn_id

  audit_log_config {
    cloudwatch_region            = "us-east-1"
    cloudwatch_group_name        = "hcp-vault-audit"
    cloudwatch_stream_name       = "secrets"
    cloudwatch_access_key_id     = var.cloudwatch_access_key_id
    cloudwatch_secret_access_key = var.cloudwatch_secret_access_key
  }
}
