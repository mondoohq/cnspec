# Compliant (out of scope): cluster does not log to S3, so log-encryption
# via a security configuration is not required.
resource "aws_emr_cluster" "pass_example" {
  name          = "no-logging-cluster"
  release_label = "emr-6.10.0"
  service_role  = "EMR_DefaultRole"
}
