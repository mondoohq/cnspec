# Non-compliant: NFS file share not KMS encrypted (defaults to false, unset).
resource "aws_storagegateway_nfs_file_share" "plain" {
  client_list  = ["10.0.0.0/8"]
  gateway_arn  = aws_storagegateway_gateway.example.arn
  location_arn = aws_s3_bucket.example.arn
  role_arn     = aws_iam_role.example.arn
}
