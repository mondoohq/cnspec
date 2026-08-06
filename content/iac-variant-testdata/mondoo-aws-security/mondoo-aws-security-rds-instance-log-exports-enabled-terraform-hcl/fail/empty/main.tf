# Non-compliant: the log export list is declared but empty.
resource "aws_db_instance" "fail_empty" {
  identifier        = "fail-empty"
  engine            = "mysql"
  instance_class    = "db.t3.medium"
  allocated_storage = 20

  enabled_cloudwatch_logs_exports = []
}
