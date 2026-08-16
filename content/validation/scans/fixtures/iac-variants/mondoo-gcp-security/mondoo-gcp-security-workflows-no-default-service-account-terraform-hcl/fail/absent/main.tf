# Non-compliant: no service_account set, so the default Compute SA is used.
resource "google_workflows_workflow" "example" {
  name   = "implicit-default-workflow"
  region = "us-central1"

  source_contents = <<-SRC
    main:
      steps:
        - hello:
            return: "world"
  SRC
}
