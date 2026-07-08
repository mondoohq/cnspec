# Non-compliant: workflow uses the default Compute Engine service account.
resource "google_workflows_workflow" "example" {
  name            = "default-sa-workflow"
  region          = "us-central1"
  service_account = "123456789012-compute@developer.gserviceaccount.com"

  source_contents = <<-SRC
    main:
      steps:
        - hello:
            return: "world"
  SRC
}
