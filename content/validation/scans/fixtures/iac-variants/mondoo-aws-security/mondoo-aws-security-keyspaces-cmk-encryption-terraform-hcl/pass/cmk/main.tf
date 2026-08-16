# Compliant: customer managed KMS key encryption.
resource "aws_keyspaces_table" "pass_example" {
  keyspace_name = "example"
  table_name    = "pass_example"

  schema_definition {
    column {
      name = "id"
      type = "text"
    }
    partition_key {
      name = "id"
    }
  }

  encryption_specification {
    type = "CUSTOMER_MANAGED_KMS_KEY"
    kms_key_identifier = "arn:aws:kms:us-east-1:123456789012:key/abcd-1234"
  }
}
