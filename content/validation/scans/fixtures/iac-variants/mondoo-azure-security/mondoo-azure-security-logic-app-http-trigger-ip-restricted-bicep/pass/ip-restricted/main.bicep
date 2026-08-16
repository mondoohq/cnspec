resource workflow 'Microsoft.Logic/workflows@2019-05-01' = {
  name: 'order-processing'
  location: 'eastus'
  properties: {
    state: 'Enabled'
    accessControl: {
      triggers: {
        allowedCallerIpAddresses: [
          {
            addressRange: '13.86.221.220/30'
          }
          {
            addressRange: '40.76.0.0/24'
          }
        ]
      }
    }
    definition: {
      '$schema': 'https://schema.management.azure.com/providers/Microsoft.Logic/schemas/2016-06-01/workflowdefinition.json#'
      contentVersion: '1.0.0.0'
      triggers: {}
      actions: {}
    }
  }
}
