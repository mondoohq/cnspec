# Compliant: managed master password is encrypted with a customer-managed KMS key.
resource "aws_redshift_cluster" "pass_example" {
  cluster_identifier                = "example-cluster"
  node_type                         = "ra3.xlplus"
  master_username                   = "admin"
  manage_master_password            = true
  master_password_secret_kms_key_id = "arn:aws:kms:us-east-1:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab"
}
