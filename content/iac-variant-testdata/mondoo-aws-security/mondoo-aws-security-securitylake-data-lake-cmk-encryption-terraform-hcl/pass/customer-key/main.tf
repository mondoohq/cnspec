resource "aws_securitylake_data_lake" "primary" {
  meta_store_manager_role_arn = aws_iam_role.securitylake.arn

  configuration {
    region = "us-east-1"

    encryption_configuration {
      kms_key_id = aws_kms_key.securitylake.id
    }

    lifecycle_configuration {
      transition {
        days          = 31
        storage_class = "STANDARD_IA"
      }
    }
  }
}
