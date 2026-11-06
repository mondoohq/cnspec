# Code interpreter sessions run with restricted network access.
resource "aws_bedrockagentcore_code_interpreter" "example" {
  name = "analysis_interpreter"

  network_configuration {
    network_mode = "SANDBOX"
  }
}
