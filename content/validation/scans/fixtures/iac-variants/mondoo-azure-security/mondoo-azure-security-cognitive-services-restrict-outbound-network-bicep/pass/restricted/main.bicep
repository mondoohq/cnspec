resource vision 'Microsoft.CognitiveServices/accounts@2023-05-01' = {
  name: 'vision'
  location: 'eastus'
  kind: 'ComputerVision'
  sku: {
    name: 'S1'
  }
  properties: {
    restrictOutboundNetworkAccess: true
    allowedFqdnList: [
      'storage.example.com'
    ]
  }
}
