resource ruleGroup 'Microsoft.Network/firewallPolicies/ruleCollectionGroups@2023-09-01' = {
  name: 'prod-policy/prod-nat'
  properties: {
    priority: 300
    ruleCollections: [
      {
        name: 'inbound-web'
        ruleCollectionType: 'FirewallPolicyNatRuleCollection'
        priority: 300
        action: {
          type: 'Dnat'
        }
        rules: [
          {
            name: 'web'
            ruleType: 'NatRule'
            ipProtocols: [
              'TCP'
            ]
            sourceAddresses: [
              '*'
            ]
            destinationAddresses: [
              '203.0.113.10'
            ]
            destinationPorts: [
              '443'
            ]
            translatedAddress: '10.0.1.10'
            translatedPort: '8443'
          }
        ]
      }
    ]
  }
}
