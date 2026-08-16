# Non-compliant: RDS cluster instance is publicly accessible.
resource "aws_rds_cluster_instance" "fail_example" {
  identifier          = "example-instance"
  cluster_identifier  = "example"
  instance_class      = "db.r5.large"
  engine              = "aurora-mysql"
  publicly_accessible = true
}
