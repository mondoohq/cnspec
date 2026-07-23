# Non-compliant: default permissions granted to IAM_ALLOWED_PRINCIPALS.
resource "aws_lakeformation_data_lake_settings" "fail_example" {
  admins = ["arn:aws:iam::111122223333:user/admin"]

  create_table_default_permissions {
    principal   = "IAM_ALLOWED_PRINCIPALS"
    permissions = ["ALL"]
  }

  create_database_default_permissions {
    principal   = "IAM_ALLOWED_PRINCIPALS"
    permissions = ["ALL"]
  }
}
