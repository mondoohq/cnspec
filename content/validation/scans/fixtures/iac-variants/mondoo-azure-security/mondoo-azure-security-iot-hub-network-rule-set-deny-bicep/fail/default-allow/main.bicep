// Allow-by-default means the IP rules constrain nothing: every address not
// explicitly listed still reaches the hub.
resource iotHub 'Microsoft.Devices/IotHubs@2023-06-30' = {
  name: 'fleet-hub'
  location: 'eastus'
  sku: {
    name: 'S1'
    capacity: 1
  }
  properties: {
    networkRuleSets: {
      defaultAction: 'Allow'
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
