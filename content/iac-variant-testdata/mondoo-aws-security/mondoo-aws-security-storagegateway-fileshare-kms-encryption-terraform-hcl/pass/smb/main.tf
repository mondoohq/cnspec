# Compliant: SMB file share encrypted with a KMS key.
resource "aws_storagegateway_smb_file_share" "enc" {
  authentication = "ActiveDirectory"
  gateway_arn    = aws_storagegateway_gateway.example.arn
  location_arn   = aws_s3_bucket.example.arn
  role_arn       = aws_iam_role.example.arn

  kms_encrypted = true
  kms_key_arn   = aws_kms_key.example.arn
}
