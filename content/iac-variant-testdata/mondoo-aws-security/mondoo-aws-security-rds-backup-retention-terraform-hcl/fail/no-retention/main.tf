# Non-compliant: DB instance disables automated backups.
resource "aws_db_instance" "fail_example" {
  allocated_storage       = 20
  engine                  = "mysql"
  instance_class          = "db.t3.micro"
  backup_retention_period = 0
}
