# Non-compliant: SMB file share explicitly disables KMS encryption.
resource "aws_storagegateway_smb_file_share" "plain" {
  authentication = "ActiveDirectory"
  gateway_arn    = aws_storagegateway_gateway.example.arn
  location_arn   = aws_s3_bucket.example.arn
  role_arn       = aws_iam_role.example.arn

  kms_encrypted = false
}
