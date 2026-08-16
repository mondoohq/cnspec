# Compliant: call_log_level logs errors only.
resource "google_workflows_workflow" "example" {
  name           = "errors-workflow"
  region         = "us-central1"
  call_log_level = "LOG_ERRORS_ONLY"

  source_contents = <<-SRC
    main:
      steps:
        - hello:
            return: "world"
  SRC
}
