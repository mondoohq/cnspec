# Compliant: DMS replication instance uses a customer-managed KMS key.
resource "aws_dms_replication_instance" "pass_example" {
  replication_instance_id    = "example-instance"
  replication_instance_class = "dms.t3.micro"
  kms_key_arn                = "arn:aws:kms:us-east-1:123456789012:key/abcd1234-a123-456a-a12b-a123b4cd56ef"
}
