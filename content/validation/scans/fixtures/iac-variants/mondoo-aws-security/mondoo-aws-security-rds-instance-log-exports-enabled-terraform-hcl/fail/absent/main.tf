# Non-compliant: no log types are exported, so engine logs stay on the instance.
resource "aws_db_instance" "fail_absent" {
  identifier        = "fail-absent"
  engine            = "postgres"
  instance_class    = "db.t3.medium"
  allocated_storage = 20
}
