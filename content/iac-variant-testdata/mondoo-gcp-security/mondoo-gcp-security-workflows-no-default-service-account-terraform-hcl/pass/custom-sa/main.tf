# Compliant: workflow runs as a dedicated custom service account.
resource "google_workflows_workflow" "example" {
  name            = "least-priv-workflow"
  region          = "us-central1"
  service_account = "workflow-runner@my-project.iam.gserviceaccount.com"

  source_contents = <<-SRC
    main:
      steps:
        - hello:
            return: "world"
  SRC
}
