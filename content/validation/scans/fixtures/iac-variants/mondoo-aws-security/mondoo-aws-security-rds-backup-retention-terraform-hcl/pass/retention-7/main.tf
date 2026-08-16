# Compliant: DB instance retains backups for 7 days.
resource "aws_db_instance" "pass_example" {
  allocated_storage       = 20
  engine                  = "mysql"
  instance_class          = "db.t3.micro"
  backup_retention_period = 7
}
