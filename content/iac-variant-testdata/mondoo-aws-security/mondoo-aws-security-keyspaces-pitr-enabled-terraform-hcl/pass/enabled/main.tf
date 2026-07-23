# Compliant: point-in-time recovery enabled.
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

  point_in_time_recovery {
    status = "ENABLED"
  }
}
