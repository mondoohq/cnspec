resource "aws_neptune_cluster" "example" {
  cluster_identifier = "example"
  storage_encrypted  = true
}
