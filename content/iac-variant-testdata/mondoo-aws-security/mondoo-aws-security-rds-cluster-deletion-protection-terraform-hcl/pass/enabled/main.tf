# Compliant: cluster enables deletion protection.
resource "aws_rds_cluster" "pass_example" {
  cluster_identifier  = "example"
  engine              = "aurora-mysql"
  deletion_protection = true
}
