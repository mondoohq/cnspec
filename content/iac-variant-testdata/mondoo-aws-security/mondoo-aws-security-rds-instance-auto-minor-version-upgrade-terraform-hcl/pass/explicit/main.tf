# Compliant: RDS DB instance explicitly enables automatic minor version upgrades.
resource "aws_db_instance" "pass_explicit" {
  identifier                 = "pass-explicit"
  engine                     = "postgres"
  instance_class             = "db.t3.medium"
  allocated_storage          = 20
  auto_minor_version_upgrade = true
}
