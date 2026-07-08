# Non-compliant: DB instance is encrypted but uses no customer-managed KMS key.
resource "aws_db_instance" "fail_example" {
  identifier        = "example"
  engine            = "mysql"
  instance_class    = "db.t3.micro"
  allocated_storage = 20
  storage_encrypted = true
}
