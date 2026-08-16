# Compliant: FSx for Lustre file system encrypted with a KMS key.
resource "aws_fsx_lustre_file_system" "pass_example" {
  storage_capacity            = 1200
  subnet_ids                  = ["subnet-0123456789abcdef0"]
  deployment_type             = "PERSISTENT_1"
  per_unit_storage_throughput = 50
  kms_key_id                  = "arn:aws:kms:us-east-1:111122223333:key/abcd-1234"
}
