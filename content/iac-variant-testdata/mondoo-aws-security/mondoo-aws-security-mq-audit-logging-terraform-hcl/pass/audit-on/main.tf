# Compliant: MQ broker has audit logging enabled.
resource "aws_mq_broker" "pass_example" {
  broker_name = "pass-example"
  engine_type = "ActiveMQ"

  logs {
    audit   = true
    general = true
  }
}
