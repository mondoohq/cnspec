# Non-compliant: require_tls is omitted, so the proxy defaults to not requiring TLS.
resource "aws_db_proxy" "example" {
  name           = "example"
  engine_family  = "MYSQL"
  role_arn       = "arn:aws:iam::123456789012:role/example"
  vpc_subnet_ids = ["subnet-1234"]

  auth {
    auth_scheme = "SECRETS"
    iam_auth    = "REQUIRED"
    secret_arn  = "arn:aws:secretsmanager:us-east-1:123456789012:secret:example"
  }
}
