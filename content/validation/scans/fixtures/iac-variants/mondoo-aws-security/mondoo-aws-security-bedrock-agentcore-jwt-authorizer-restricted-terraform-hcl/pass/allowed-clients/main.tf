# The JWT authorizer only accepts tokens issued to the listed client.
resource "aws_bedrockagentcore_gateway" "example" {
  name            = "tools-gateway"
  role_arn        = "arn:aws:iam::111122223333:role/gateway"
  authorizer_type = "CUSTOM_JWT"

  authorizer_configuration {
    custom_jwt_authorizer {
      discovery_url   = "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_example/.well-known/openid-configuration"
      allowed_clients = ["example-client-id"]
    }
  }
}
