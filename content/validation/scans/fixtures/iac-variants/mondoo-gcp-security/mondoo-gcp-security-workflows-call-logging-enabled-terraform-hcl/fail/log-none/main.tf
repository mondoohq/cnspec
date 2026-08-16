# Non-compliant: call logging explicitly disabled.
resource "google_workflows_workflow" "example" {
  name           = "silent-workflow"
  region         = "us-central1"
  call_log_level = "LOG_NONE"

  source_contents = <<-SRC
    main:
      steps:
        - hello:
            return: "world"
  SRC
}
