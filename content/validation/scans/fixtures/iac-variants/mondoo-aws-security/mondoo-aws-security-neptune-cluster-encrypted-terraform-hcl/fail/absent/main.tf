# Non-compliant: storage_encrypted is omitted, so it defaults to false (unencrypted).
resource "aws_neptune_cluster" "fail_example" {
  cluster_identifier = "example"
  engine             = "neptune"
}
