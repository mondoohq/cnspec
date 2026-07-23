# Compliant: publicly_accessible is omitted, so both default to private.
resource "aws_rds_cluster" "pass_example" {
  cluster_identifier = "example"
  engine             = "aurora-mysql"
}

resource "aws_rds_cluster_instance" "pass_example" {
  identifier         = "example-instance"
  cluster_identifier = aws_rds_cluster.pass_example.id
  instance_class     = "db.r5.large"
  engine             = "aurora-mysql"
}
