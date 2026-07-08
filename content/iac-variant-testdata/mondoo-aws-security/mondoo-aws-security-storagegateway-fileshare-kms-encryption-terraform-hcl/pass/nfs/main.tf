# Compliant: NFS file share encrypted with a KMS key.
resource "aws_storagegateway_nfs_file_share" "enc" {
  client_list  = ["10.0.0.0/8"]
  gateway_arn  = aws_storagegateway_gateway.example.arn
  location_arn = aws_s3_bucket.example.arn
  role_arn     = aws_iam_role.example.arn

  kms_encrypted = true
  kms_key_arn   = aws_kms_key.example.arn
}
