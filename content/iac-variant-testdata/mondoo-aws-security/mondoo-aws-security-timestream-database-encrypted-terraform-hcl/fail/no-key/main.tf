# Non-compliant: no KMS key specified for the Timestream database.
resource "aws_timestreamwrite_database" "plain" {
  database_name = "metrics"
}
