# Non-compliant: cluster disables deletion protection.
resource "aws_rds_cluster" "fail_example" {
  cluster_identifier  = "example"
  engine              = "aurora-mysql"
  deletion_protection = false
}
