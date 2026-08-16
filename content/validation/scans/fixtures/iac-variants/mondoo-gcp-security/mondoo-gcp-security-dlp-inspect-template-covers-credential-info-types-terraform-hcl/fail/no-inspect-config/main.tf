# Non-compliant: inspect template has no inspect_config block at all.
resource "google_data_loss_prevention_inspect_template" "empty" {
  parent       = "projects/my-project/locations/us-central1"
  display_name = "empty-scanner"
  description  = "placeholder template with no inspect configuration"
}
