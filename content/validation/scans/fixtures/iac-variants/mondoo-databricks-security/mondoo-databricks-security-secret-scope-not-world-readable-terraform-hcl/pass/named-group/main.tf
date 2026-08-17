resource "databricks_secret_acl" "prod" {
  principal  = "data-engineering"
  permission = "READ"
  scope      = "prod"
}
