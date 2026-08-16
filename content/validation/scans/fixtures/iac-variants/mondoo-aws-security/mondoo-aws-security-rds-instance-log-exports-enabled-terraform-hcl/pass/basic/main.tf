# Compliant: RDS DB instance publishes PostgreSQL engine logs to CloudWatch Logs.
resource "aws_db_instance" "pass_example" {
  identifier        = "pass-example"
  engine            = "postgres"
  instance_class    = "db.t3.medium"
  allocated_storage = 20

  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]
}
