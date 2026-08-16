# Compliant: cluster has an associated logging resource.
resource "aws_redshift_cluster" "example" {
  cluster_identifier = "example"
  node_type          = "dc2.large"
  master_username    = "admin"
}

resource "aws_redshift_logging" "example" {
  cluster_identifier = aws_redshift_cluster.example.id
  log_destination_type = "cloudwatch"
}
