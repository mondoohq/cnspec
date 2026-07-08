# Non-compliant: RDS cluster is publicly accessible.
resource "aws_rds_cluster" "fail_example" {
  cluster_identifier  = "example"
  engine              = "aurora-mysql"
  publicly_accessible = true
}
