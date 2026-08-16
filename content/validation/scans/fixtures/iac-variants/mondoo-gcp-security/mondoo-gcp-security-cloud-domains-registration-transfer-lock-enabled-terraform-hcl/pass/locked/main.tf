resource "google_clouddomains_registration" "corp" {
  project     = var.project_id
  location    = "global"
  domain_name = "example-corp.com"

  yearly_price {
    currency_code = "USD"
    units         = 12
  }

  management_settings {
    transfer_lock_state = "LOCKED"
  }

  dns_settings {
    custom_dns {
      name_servers = ["ns-cloud-a1.googledomains.com", "ns-cloud-a2.googledomains.com"]
    }
  }

  contact_settings {
    privacy = "PRIVATE_CONTACT_DATA"

    registrant_contact {
      email = "domains@example-corp.com"
      phone_number = "+1.5555550100"

      postal_address {
        region_code  = "US"
        postal_code  = "94105"
        address_lines = ["1 Example Way"]
        locality      = "San Francisco"
        recipients    = ["Example Corp"]
      }
    }
  }
}
