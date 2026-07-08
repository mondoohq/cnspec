# Compliant: IAM database authentication is enabled.
resource "aws_neptune_cluster" "pass_example" {
  cluster_identifier                  = "example"
  iam_database_authentication_enabled = true
}
