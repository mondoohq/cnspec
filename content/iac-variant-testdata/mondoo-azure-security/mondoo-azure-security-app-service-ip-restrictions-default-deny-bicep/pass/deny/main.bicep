resource siteConfig 'Microsoft.Web/sites/config@2022-09-01' = {
  name: 'app-prod-001/web'
  properties: {
    ipSecurityRestrictionsDefaultAction: 'Deny'
    ipSecurityRestrictions: [
      {
        ipAddress: '203.0.113.0/24'
        action: 'Allow'
        priority: 100
        name: 'corp-network'
      }
    ]
  }
}
