resource functionApp 'Microsoft.Web/sites@2023-01-01' = {
  name: 'event-processor'
  location: 'eastus'
  kind: 'functionapp'
  properties: {
    siteConfig: {
      ipSecurityRestrictionsDefaultAction: 'Deny'
      ipSecurityRestrictions: [
        {
          name: 'corporate-egress'
          action: 'Allow'
          ipAddress: '203.0.113.0/24'
          priority: 100
        }
      ]
    }
  }
}
