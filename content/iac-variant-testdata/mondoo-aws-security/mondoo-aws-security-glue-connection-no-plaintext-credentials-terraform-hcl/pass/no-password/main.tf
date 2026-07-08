# Compliant: Glue connection has no plaintext PASSWORD in connection_properties.
resource "aws_glue_connection" "pass_example" {
  name = "example-connection"
  connection_properties = {
    JDBC_CONNECTION_URL = "jdbc:mysql://example.internal:3306/db"
    USERNAME            = "app_user"
  }
}
