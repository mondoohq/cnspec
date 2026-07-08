# Non-compliant: DB instance is publicly accessible.
resource "aws_db_instance" "fail_example" {
  identifier          = "example"
  engine              = "mysql"
  instance_class      = "db.t3.micro"
  allocated_storage   = 20
  publicly_accessible = true
}
