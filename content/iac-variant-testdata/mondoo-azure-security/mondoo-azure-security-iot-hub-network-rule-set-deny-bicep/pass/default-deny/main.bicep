resource iotHub 'Microsoft.Devices/IotHubs@2023-06-30' = {
  name: 'fleet-hub'
  location: 'eastus'
  sku: {
    name: 'S1'
    capacity: 1
  }
  properties: {
    networkRuleSets: {
      defaultAction: 'Deny'
      applyToBuiltInEventHubEndpoint: true
      ipRules: [
        {
          filterName: 'corporate-egress'
          action: 'Allow'
          ipMask: '203.0.113.0/24'
        }
      ]
    }
  }
}
