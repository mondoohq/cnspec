resource api 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'api'
  location: 'eastus'
  properties: {
    configuration: {
      ingress: {
        external: true
        targetPort: 8080
        ipSecurityRestrictions: [
          {
            name: 'corporate-egress'
            action: 'Allow'
            ipAddressRange: '203.0.113.0/24'
          }
        ]
      }
    }
  }
}
