# Compliant: RDS cluster and instance are not publicly accessible.
resource "aws_rds_cluster" "pass_example" {
  cluster_identifier  = "example"
  engine              = "aurora-mysql"
  publicly_accessible = false
}

resource "aws_rds_cluster_instance" "pass_example" {
  identifier          = "example-instance"
  cluster_identifier  = aws_rds_cluster.pass_example.id
  instance_class      = "db.r5.large"
  engine              = "aurora-mysql"
  publicly_accessible = false
}
