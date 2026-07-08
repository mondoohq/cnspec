# Non-compliant: cluster is not encrypted and has no customer managed key.
resource "aws_rds_cluster" "fail_example" {
  cluster_identifier = "example"
  engine             = "aurora-mysql"
  storage_encrypted  = false
}
