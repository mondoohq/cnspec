# Non-compliant: cluster logs to S3 with a security configuration but no log
# encryption key; the security configuration does not set the log key.
resource "aws_emr_cluster" "fail_example" {
  name                   = "fail-cluster"
  release_label          = "emr-6.10.0"
  service_role           = "EMR_DefaultRole"
  log_uri                = "s3://example-logs/emr/"
  security_configuration = "my-sec-config"
}
