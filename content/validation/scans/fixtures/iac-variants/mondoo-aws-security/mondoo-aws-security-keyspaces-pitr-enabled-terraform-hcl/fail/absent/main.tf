# Non-compliant: no point_in_time_recovery block, so PITR stays disabled by default.
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
