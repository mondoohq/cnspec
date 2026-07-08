# Compliant: DMS endpoint enforces SSL.
resource "aws_dms_endpoint" "pass_example" {
  endpoint_id   = "example-endpoint"
  endpoint_type = "source"
  engine_name   = "aurora"
  ssl_mode      = "require"
}
