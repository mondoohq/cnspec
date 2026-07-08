# Non-compliant: DMS endpoint disables SSL.
resource "aws_dms_endpoint" "fail_example" {
  endpoint_id   = "example-endpoint"
  endpoint_type = "source"
  engine_name   = "aurora"
  ssl_mode      = "none"
}
