resource "databricks_mws_workspaces" "this" {
  account_id                      = var.account_id
  workspace_name                  = "prod"
  storage_customer_managed_key_id = databricks_mws_customer_managed_keys.storage.customer_managed_key_id
}
