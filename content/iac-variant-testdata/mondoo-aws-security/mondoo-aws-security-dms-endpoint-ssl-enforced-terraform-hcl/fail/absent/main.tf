# Non-compliant: DMS endpoint omits ssl_mode, so it defaults to "none" (no SSL).
resource "aws_dms_endpoint" "fail_example" {
  endpoint_id   = "example-endpoint"
  endpoint_type = "source"
  engine_name   = "aurora"
}
