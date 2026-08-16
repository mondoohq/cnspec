# Non-compliant: RDS DB instance explicitly disables automatic minor version upgrades.
resource "aws_db_instance" "fail_disabled" {
  identifier                 = "fail-disabled"
  engine                     = "postgres"
  instance_class             = "db.t3.medium"
  allocated_storage          = 20
  auto_minor_version_upgrade = false
}
