# Non-compliant: DB instance has Performance Insights disabled.
resource "aws_db_instance" "fail_example" {
  identifier                   = "example"
  engine                       = "mysql"
  instance_class               = "db.t3.micro"
  allocated_storage            = 20
  performance_insights_enabled = false
}
