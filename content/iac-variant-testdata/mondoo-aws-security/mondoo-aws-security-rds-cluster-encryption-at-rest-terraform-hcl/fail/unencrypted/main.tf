# Non-compliant: cluster does not enable storage encryption at rest.
resource "aws_rds_cluster" "fail_example" {
  cluster_identifier = "example"
  engine             = "aurora-mysql"
  storage_encrypted  = false
}
