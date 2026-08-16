# Compliant: workflow encrypted with a customer-managed KMS key.
resource "google_workflows_workflow" "example" {
  name            = "encrypted-workflow"
  region          = "us-central1"
  crypto_key_name = "projects/my-project/locations/us-central1/keyRings/wf-ring/cryptoKeys/wf-key"

  source_contents = <<-SRC
    main:
      steps:
        - hello:
            return: "world"
  SRC
}
