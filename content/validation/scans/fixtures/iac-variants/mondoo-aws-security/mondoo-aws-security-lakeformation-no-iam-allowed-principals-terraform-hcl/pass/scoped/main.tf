# Compliant: default permissions granted to a specific principal, not IAM_ALLOWED_PRINCIPALS.
resource "aws_lakeformation_data_lake_settings" "pass_example" {
  admins = ["arn:aws:iam::111122223333:user/admin"]

  create_table_default_permissions {
    principal   = "arn:aws:iam::111122223333:role/data-role"
    permissions = ["SELECT"]
  }

  create_database_default_permissions {
    principal   = "arn:aws:iam::111122223333:role/data-role"
    permissions = ["ALL"]
  }
}
