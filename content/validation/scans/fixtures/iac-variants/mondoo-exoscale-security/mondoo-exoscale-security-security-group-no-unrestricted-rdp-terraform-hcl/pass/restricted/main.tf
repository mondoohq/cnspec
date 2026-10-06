resource "exoscale_security_group" "web" {
  name = "web"
}
resource "exoscale_security_group_rule" "r" {
  security_group_id = exoscale_security_group.web.id
  type              = "INGRESS"
  protocol          = "TCP"
  cidr              = "203.0.113.0/24"
  start_port        = 3389
  end_port          = 3389
}
