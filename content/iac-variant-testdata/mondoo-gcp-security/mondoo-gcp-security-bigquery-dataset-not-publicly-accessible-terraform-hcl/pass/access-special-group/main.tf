# Compliant: access blocks use non-public special groups and named principals.
resource "google_bigquery_dataset" "scoped" {
  dataset_id = "my_dataset"
  location   = "US"

  access {
    role          = "OWNER"
    special_group = "projectOwners"
  }

  access {
    role          = "READER"
    user_by_email = "analyst@example.com"
  }
}
