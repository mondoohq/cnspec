// A bypass rule exempts traffic from intrusion detection entirely. Broad
// bypasses are how IDPS ends up enabled but blind on the paths that matter.
resource firewallPolicy 'Microsoft.Network/firewallPolicies@2023-09-01' = {
  name: 'prod-policy'
  location: 'eastus'
  properties: {
    sku: {
      tier: 'Premium'
    }
    intrusionDetection: {
      mode: 'Deny'
      configuration: {
        bypassTrafficSettings: [
          {
            name: 'skip-partner-traffic'
            protocol: 'TCP'
            sourceAddresses: [
              '10.0.0.0/8'
            ]
            destinationPorts: [
              '443'
            ]
          }
        ]
      }
    }
  }
}
