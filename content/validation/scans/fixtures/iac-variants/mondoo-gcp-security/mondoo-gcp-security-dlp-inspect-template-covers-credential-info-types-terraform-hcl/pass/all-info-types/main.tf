# Compliant: inspect template covers all four credential info types.
resource "google_data_loss_prevention_inspect_template" "compliant" {
  parent       = "projects/my-project/locations/us-central1"
  display_name = "credential-scanner"

  inspect_config {
    info_types {
      name = "AWS_CREDENTIALS"
    }
    info_types {
      name = "GCP_CREDENTIALS"
    }
    info_types {
      name = "JSON_WEB_TOKEN"
    }
    info_types {
      name = "BASIC_AUTH_HEADER"
    }

    min_likelihood = "POSSIBLE"
  }
}
