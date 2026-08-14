// Microsoft.CognitiveServices is the service-managed key source, so uploaded
// training and inference content is not under customer key control.
resource vision 'Microsoft.CognitiveServices/accounts@2023-05-01' = {
  name: 'vision'
  location: 'eastus'
  kind: 'ComputerVision'
  sku: {
    name: 'S1'
  }
  properties: {
    encryption: {
      keySource: 'Microsoft.CognitiveServices'
    }
  }
}
