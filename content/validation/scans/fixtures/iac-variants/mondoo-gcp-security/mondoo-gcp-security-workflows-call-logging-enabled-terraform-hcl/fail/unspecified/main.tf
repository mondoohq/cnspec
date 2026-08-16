# Non-compliant: call_log_level unspecified.
resource "google_workflows_workflow" "example" {
  name           = "unspecified-workflow"
  region         = "us-central1"
  call_log_level = "CALL_LOG_LEVEL_UNSPECIFIED"

  source_contents = <<-SRC
    main:
      steps:
        - hello:
            return: "world"
  SRC
}
