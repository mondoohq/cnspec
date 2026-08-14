# S3_MANAGED_KEY is the service default. The security data lake aggregates
# findings and logs from the whole org, so it is exactly the store that should
# be under a customer-controlled key.
resource "aws_securitylake_data_lake" "primary" {
  meta_store_manager_role_arn = aws_iam_role.securitylake.arn

  configuration {
    region = "us-east-1"

    encryption_configuration {
      kms_key_id = "S3_MANAGED_KEY"
    }
  }
}
