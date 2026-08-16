# Non-compliant: no contact subscribes to SECURITY or ALL categories.
resource "google_essential_contacts_contact" "technical" {
  parent                              = "organizations/123456789"
  email                               = "ops@example.com"
  language_tag                        = "en-US"
  notification_category_subscriptions = ["TECHNICAL"]
}

resource "google_essential_contacts_contact" "billing" {
  parent                              = "organizations/123456789"
  email                               = "billing@example.com"
  language_tag                        = "en-US"
  notification_category_subscriptions = ["BILLING", "PRODUCT_UPDATES"]
}
