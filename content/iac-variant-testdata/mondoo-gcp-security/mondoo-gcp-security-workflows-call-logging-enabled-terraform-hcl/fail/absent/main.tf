# Non-compliant: call_log_level not set (defaults to unspecified / no logging).
resource "google_workflows_workflow" "example" {
  name   = "default-workflow"
  region = "us-central1"

  source_contents = <<-SRC
    main:
      steps:
        - hello:
            return: "world"
  SRC
}
