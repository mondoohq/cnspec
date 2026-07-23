# Compliant: publicly_accessible is omitted, so the instance defaults to private.
resource "aws_db_instance" "pass_example" {
  identifier        = "example"
  engine            = "mysql"
  instance_class    = "db.t3.micro"
  allocated_storage = 20
}
