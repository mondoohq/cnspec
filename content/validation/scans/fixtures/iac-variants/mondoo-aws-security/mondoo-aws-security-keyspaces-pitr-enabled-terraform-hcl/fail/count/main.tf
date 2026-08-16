# Non-compliant: a counted keyspaces table with point-in-time recovery DISABLED.
resource "aws_keyspaces_table" "counted" {
  count = 2

  keyspace_name = "example"
  table_name    = "counted_${count.index}"

  schema_definition {
    column {
      name = "id"
      type = "text"
    }
    partition_key {
      name = "id"
    }
  }

  point_in_time_recovery {
    status = "DISABLED"
  }
}
