# Non-compliant: no crypto_key_name; Google-managed encryption only.
resource "google_workflows_workflow" "example" {
  name   = "unencrypted-workflow"
  region = "us-central1"

  source_contents = <<-SRC
    main:
      steps:
        - hello:
            return: "world"
  SRC
}
