# Non-compliant: inspect template omits BASIC_AUTH_HEADER (and others).
resource "google_data_loss_prevention_inspect_template" "partial" {
  parent       = "projects/my-project/locations/us-central1"
  display_name = "credential-scanner"

  inspect_config {
    info_types {
      name = "AWS_CREDENTIALS"
    }
    info_types {
      name = "GCP_CREDENTIALS"
    }

    min_likelihood = "POSSIBLE"
  }
}
