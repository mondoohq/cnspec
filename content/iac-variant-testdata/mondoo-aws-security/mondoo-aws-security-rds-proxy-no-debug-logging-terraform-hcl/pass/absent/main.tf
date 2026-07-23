# Compliant: debug_logging is omitted, so it defaults to disabled.
resource "aws_db_proxy" "pass_example" {
  name           = "example"
  engine_family  = "MYSQL"
  role_arn       = "arn:aws:iam::123456789012:role/example"
  vpc_subnet_ids = ["subnet-12345678"]

  auth {
    auth_scheme = "SECRETS"
    iam_auth    = "REQUIRED"
    secret_arn  = "arn:aws:secretsmanager:us-east-1:123456789012:secret:example"
  }
}
