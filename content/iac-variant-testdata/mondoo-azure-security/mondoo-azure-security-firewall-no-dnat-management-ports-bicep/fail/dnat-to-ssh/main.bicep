// DNAT publishing SSH straight to an internal host puts a management port on
// the public internet behind a single NAT rule.
resource ruleGroup 'Microsoft.Network/firewallPolicies/ruleCollectionGroups@2023-09-01' = {
  name: 'prod-nat'
  properties: {
    priority: 300
    ruleCollections: [
      {
        name: 'inbound-admin'
        ruleCollectionType: 'FirewallPolicyNatRuleCollection'
        priority: 300
        action: {
          type: 'Dnat'
        }
        rules: [
          {
            name: 'ssh-jump'
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
              '2222'
            ]
            translatedAddress: '10.0.1.5'
            translatedPort: '22'
          }
        ]
      }
    ]
  }
}
