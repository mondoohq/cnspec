# Compliant: call_log_level logs all calls.
resource "google_workflows_workflow" "example" {
  name            = "audited-workflow"
  region          = "us-central1"
  service_account = "workflow-sa@my-project.iam.gserviceaccount.com"
  call_log_level  = "LOG_ALL_CALLS"

  source_contents = <<-SRC
    main:
      steps:
        - hello:
            return: "world"
  SRC
}
