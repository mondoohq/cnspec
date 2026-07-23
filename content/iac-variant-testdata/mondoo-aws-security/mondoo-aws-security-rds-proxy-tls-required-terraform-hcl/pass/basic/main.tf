# Compliant: proxy requires TLS.
resource "aws_db_proxy" "example" {
  name           = "example"
  engine_family  = "MYSQL"
  require_tls    = true
  role_arn       = "arn:aws:iam::123456789012:role/example"
  vpc_subnet_ids = ["subnet-1234"]
}
