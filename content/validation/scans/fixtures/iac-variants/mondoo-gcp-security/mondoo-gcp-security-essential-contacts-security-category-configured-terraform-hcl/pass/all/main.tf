# Compliant: an essential contact subscribed to ALL categories includes security notifications.
resource "google_essential_contacts_contact" "all" {
  parent                              = "organizations/123456789"
  email                               = "cloud-admins@example.com"
  language_tag                        = "en-US"
  notification_category_subscriptions = ["ALL"]
}
