# Compliant: an essential contact is subscribed to the SECURITY notification category.
resource "google_essential_contacts_contact" "security" {
  parent                              = "organizations/123456789"
  email                               = "security-team@example.com"
  language_tag                        = "en-US"
  notification_category_subscriptions = ["SECURITY"]
}

resource "google_essential_contacts_contact" "billing" {
  parent                              = "organizations/123456789"
  email                               = "billing@example.com"
  language_tag                        = "en-US"
  notification_category_subscriptions = ["BILLING"]
}
