resource "databricks_secret_acl" "prod" {
  principal  = "users"
  permission = "MANAGE"
  scope      = "prod"
}
