# Compliant: access granted only to named users and groups, no domain-wide access.
resource "google_bigquery_dataset" "scoped" {
  dataset_id = "my_dataset"
  location   = "US"

  access {
    role          = "OWNER"
    user_by_email = "owner@example.com"
  }

  access {
    role           = "READER"
    group_by_email = "analysts@example.com"
  }
}
