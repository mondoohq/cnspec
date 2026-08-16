# Non-compliant: no encryption_specification block at all, so the table
# falls back to the AWS owned default key instead of a customer managed KMS key.
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
}
