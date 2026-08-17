resource "databricks_secret_acl" "prod" {
  principal  = "users"
  permission = "READ"
  scope      = "prod"
}
