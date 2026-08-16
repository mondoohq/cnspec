# BigQuery fail fixture - every BigQuery check should fail.
#
# - Dataset has no default_encryption_configuration (no CMEK).
# - Dataset grants allAuthenticatedUsers and a domain-wide entry.
# - Table has no encryption_configuration.
# - View uses Legacy SQL.

resource "google_bigquery_dataset" "analytics" {
  dataset_id = "fail_analytics_${random_id.suffix.hex}"
  location   = "US"

  # default_encryption_configuration intentionally absent

  access {
    role          = "OWNER"
    user_by_email = "fail-owner@example.com"
  }

  access {
    role          = "READER"
    special_group = "allAuthenticatedUsers"
  }

  access {
    role   = "READER"
    domain = "example.com"
  }
}

resource "google_bigquery_table" "events" {
  dataset_id          = google_bigquery_dataset.analytics.dataset_id
  table_id            = "fail_events"
  deletion_protection = false

  # encryption_configuration intentionally absent

  schema = jsonencode([
    {
      name = "id"
      type = "STRING"
    },
  ])
}

resource "google_bigquery_table" "events_view" {
  dataset_id          = google_bigquery_dataset.analytics.dataset_id
  table_id            = "fail_events_view"
  deletion_protection = false

  view {
    query          = "SELECT id FROM [project.dataset.table]"
    use_legacy_sql = true
  }
}
