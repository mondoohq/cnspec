# Non-compliant: cluster is encrypted but uses the AWS-managed key (no CMK).
resource "aws_rds_cluster" "fail_example" {
  cluster_identifier = "example"
  engine             = "aurora-mysql"
  storage_encrypted  = true
}
