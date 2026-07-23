# Compliant: DB instance has Performance Insights enabled.
resource "aws_db_instance" "pass_example" {
  identifier                   = "example"
  engine                       = "mysql"
  instance_class               = "db.t3.micro"
  allocated_storage            = 20
  performance_insights_enabled = true
}
