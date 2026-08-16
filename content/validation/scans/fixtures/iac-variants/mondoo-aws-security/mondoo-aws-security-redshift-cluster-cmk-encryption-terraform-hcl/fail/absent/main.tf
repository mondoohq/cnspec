# Non-compliant: no kms_key_id, so encryption (if any) uses the AWS-owned default key.
resource "aws_redshift_cluster" "example" {
  cluster_identifier = "example"
  node_type          = "dc2.large"
  master_username    = "admin"
  encrypted          = true
}
