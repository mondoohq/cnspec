# Non-compliant: storage encryption is disabled.
resource "aws_neptune_cluster" "fail_example" {
  cluster_identifier = "example"
  storage_encrypted  = false
}
