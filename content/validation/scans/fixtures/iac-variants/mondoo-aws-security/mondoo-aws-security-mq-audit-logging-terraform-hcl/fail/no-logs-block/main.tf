# Non-compliant: MQ broker declares no logs block at all, so audit logging is off.
resource "aws_mq_broker" "fail_example" {
  broker_name = "fail-example"
  engine_type = "ActiveMQ"
}
