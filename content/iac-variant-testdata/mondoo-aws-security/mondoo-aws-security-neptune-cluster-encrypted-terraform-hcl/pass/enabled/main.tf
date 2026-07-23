# Compliant: storage encryption is enabled.
resource "aws_neptune_cluster" "pass_example" {
  cluster_identifier = "example"
  storage_encrypted  = true
}
