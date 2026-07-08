# Non-compliant: DB proxy enables debug logging.
resource "aws_db_proxy" "fail_example" {
  name          = "example"
  debug_logging = true
  engine_family = "MYSQL"
  role_arn      = "arn:aws:iam::123456789012:role/example"
  vpc_subnet_ids = ["subnet-12345678"]

  auth {
    auth_scheme = "SECRETS"
    iam_auth    = "REQUIRED"
    secret_arn  = "arn:aws:secretsmanager:us-east-1:123456789012:secret:example"
  }
}
