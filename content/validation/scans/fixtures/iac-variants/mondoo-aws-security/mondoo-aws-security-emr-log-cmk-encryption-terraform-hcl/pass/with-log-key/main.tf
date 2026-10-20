# Compliant: cluster logging to S3 encrypts its log files with a customer managed key.
resource "aws_emr_cluster" "pass_example" {
  name                      = "pass-cluster"
  release_label             = "emr-6.10.0"
  service_role              = "EMR_DefaultRole"
  log_uri                   = "s3://example-logs/emr/"
  log_encryption_kms_key_id = "arn:aws:kms:us-east-1:123456789012:key/abcd1234-a123-456a-a12b-a123b4cd56ef"
}
