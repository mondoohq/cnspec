# Compliant: DMS endpoint enforces SSL with full certificate verification.
resource "aws_dms_endpoint" "pass_example" {
  endpoint_id   = "example-endpoint"
  endpoint_type = "source"
  engine_name   = "aurora"
  ssl_mode      = "verify-full"
}
