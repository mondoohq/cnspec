# Compliant: DB instance is not publicly accessible.
resource "aws_db_instance" "pass_example" {
  identifier          = "example"
  engine              = "mysql"
  instance_class      = "db.t3.micro"
  allocated_storage   = 20
  publicly_accessible = false
}
