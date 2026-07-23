# Non-compliant: no encryption_spec block, so Google-managed keys are used.
resource "google_healthcare_dataset" "example" {
  name     = "example-dataset"
  location = "us-central1"
}
