# Non-compliant: MQ broker has audit logging disabled.
resource "aws_mq_broker" "fail_example" {
  broker_name = "fail-example"
  engine_type = "ActiveMQ"

  logs {
    audit   = false
    general = true
  }
}
