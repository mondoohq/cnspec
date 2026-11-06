# Model-generated code can reach the internet directly.
resource "aws_bedrockagentcore_code_interpreter" "example" {
  name = "analysis_interpreter"

  network_configuration {
    network_mode = "PUBLIC"
  }
}
