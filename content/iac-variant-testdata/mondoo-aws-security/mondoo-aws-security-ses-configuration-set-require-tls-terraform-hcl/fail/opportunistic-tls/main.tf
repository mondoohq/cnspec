# OPTIONAL means SES falls back to plaintext delivery whenever the receiving
# MTA does not offer STARTTLS, silently downgrading mail in transit.
resource "aws_sesv2_configuration_set" "transactional" {
  configuration_set_name = "transactional"

  delivery_options {
    tls_policy = "OPTIONAL"
  }
}
