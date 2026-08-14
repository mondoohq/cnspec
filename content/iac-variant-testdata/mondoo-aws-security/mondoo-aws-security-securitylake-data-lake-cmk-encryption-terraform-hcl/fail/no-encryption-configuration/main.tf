resource "aws_securitylake_data_lake" "primary" {
  meta_store_manager_role_arn = aws_iam_role.securitylake.arn

  configuration {
    region = "us-east-1"
  }
}
