# Compliant: data lake settings with no default permission blocks.
resource "aws_lakeformation_data_lake_settings" "pass_example" {
  admins                  = ["arn:aws:iam::111122223333:user/admin"]
  trusted_resource_owners = ["111122223333"]
}
