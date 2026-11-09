# The runtime accepts any token the identity provider signs.
resource "aws_bedrockagentcore_agent_runtime" "example" {
  agent_runtime_name = "support_agent"
  role_arn           = "arn:aws:iam::111122223333:role/runtime"

  network_configuration {
    network_mode = "PUBLIC"
  }

  agent_runtime_artifact {
    container_configuration {
      container_uri = "111122223333.dkr.ecr.us-east-1.amazonaws.com/agent:latest"
    }
  }

  authorizer_configuration {
    custom_jwt_authorizer {
      discovery_url = "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_example/.well-known/openid-configuration"
      allowed_scopes = ["agent/invoke"]
    }
  }
}
