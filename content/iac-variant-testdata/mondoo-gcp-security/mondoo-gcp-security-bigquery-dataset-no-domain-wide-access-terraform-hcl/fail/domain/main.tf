# Non-compliant: access granted to an entire domain.
resource "google_bigquery_dataset" "domain_wide" {
  dataset_id = "my_dataset"
  location   = "US"

  access {
    role          = "OWNER"
    user_by_email = "owner@example.com"
  }

  access {
    role   = "READER"
    domain = "example.com"
  }
}
