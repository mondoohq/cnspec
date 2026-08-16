// The service can reach any outbound host, so a prompt-injected or
// misconfigured workload can use it to exfiltrate data anywhere.
resource vision 'Microsoft.CognitiveServices/accounts@2023-05-01' = {
  name: 'vision'
  location: 'eastus'
  kind: 'ComputerVision'
  sku: {
    name: 'S1'
  }
  properties: {
    restrictOutboundNetworkAccess: false
  }
}
