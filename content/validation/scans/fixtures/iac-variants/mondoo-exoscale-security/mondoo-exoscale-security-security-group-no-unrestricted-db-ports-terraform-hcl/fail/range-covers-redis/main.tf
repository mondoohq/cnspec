resource "exoscale_security_group" "web" {
  name = "web"
}
resource "exoscale_security_group_rule" "r" {
  security_group_id = exoscale_security_group.web.id
  type              = "INGRESS"
  protocol          = "TCP"
  cidr              = "::/0"
  start_port        = 6000
  end_port          = 7000
}
