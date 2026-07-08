# Non-compliant: AWS owned key instead of customer managed KMS key.
resource "aws_keyspaces_table" "fail_example" {
  keyspace_name = "example"
  table_name    = "fail_example"

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
    type = "AWS_OWNED_KMS_KEY"
  }
}
