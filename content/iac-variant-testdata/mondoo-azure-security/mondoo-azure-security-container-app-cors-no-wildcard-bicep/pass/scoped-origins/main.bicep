resource api 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'api'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    configuration: {
      ingress: {
        external: true
        targetPort: 8080
        corsPolicy: {
          allowedOrigins: [
            'https://app.example.com'
            'https://admin.example.com'
          ]
        }
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
