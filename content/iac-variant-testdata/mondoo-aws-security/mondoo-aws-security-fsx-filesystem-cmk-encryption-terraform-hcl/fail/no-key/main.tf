# Non-compliant: FSx for Lustre file system has no kms_key_id set.
resource "aws_fsx_lustre_file_system" "fail_example" {
  storage_capacity            = 1200
  subnet_ids                  = ["subnet-0123456789abcdef0"]
  deployment_type             = "PERSISTENT_1"
  per_unit_storage_throughput = 50
}
