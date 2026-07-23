# Non-compliant: crypto_key_name set to an empty string.
resource "google_workflows_workflow" "example" {
  name            = "empty-key-workflow"
  region          = "us-central1"
  crypto_key_name = ""

  source_contents = <<-SRC
    main:
      steps:
        - hello:
            return: "world"
  SRC
}
