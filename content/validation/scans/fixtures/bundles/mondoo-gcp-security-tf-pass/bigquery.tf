# BigQuery fixture for the tf-pass test bundle.
#
# All datasets, tables, and views must use CMEK and avoid public / domain-wide
# access. Views set use_legacy_sql = false.

resource "google_bigquery_dataset" "analytics" {
  dataset_id = "analytics_${random_id.rnd.hex}"
  location   = "US"

  default_encryption_configuration {
    kms_key_name = google_kms_crypto_key.key.id
  }

  # Group-based access only - no allUsers / allAuthenticatedUsers / domain.
  access {
    role          = "OWNER"
    user_by_email = google_service_account.default.email
  }

  access {
    role           = "READER"
    group_by_email = "data-analysts@example.com"
  }
}

resource "google_bigquery_table" "events" {
  dataset_id          = google_bigquery_dataset.analytics.dataset_id
  table_id            = "events"
  deletion_protection = false

  encryption_configuration {
    kms_key_name = google_kms_crypto_key.key.id
  }

  schema = jsonencode([
    {
      name = "id"
      type = "STRING"
    },
    {
      name = "ts"
      type = "TIMESTAMP"
    },
  ])
}

resource "google_bigquery_table" "events_view" {
  dataset_id          = google_bigquery_dataset.analytics.dataset_id
  table_id            = "events_view"
  deletion_protection = false

  encryption_configuration {
    kms_key_name = google_kms_crypto_key.key.id
  }

  view {
    query          = "SELECT id, ts FROM `${google_bigquery_dataset.analytics.dataset_id}.events`"
    use_legacy_sql = false
  }
}
