# Compliant: AWS connection uses identity federation via an IAM role.
resource "google_bigquery_connection" "aws" {
  connection_id = "my-aws-connection"
  location      = "US"

  aws {
    access_role {
      iam_role_id = "arn:aws:iam::123456789012:role/bigquery-federation"
    }
  }
}
