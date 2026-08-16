# Compliant: cluster enables storage encryption at rest.
resource "aws_rds_cluster" "pass_example" {
  cluster_identifier = "example"
  engine             = "aurora-mysql"
  storage_encrypted  = true
}
