// Any source to any destination, allowed. A rule like this makes every other
// rule in the policy irrelevant.
resource ruleGroup 'Microsoft.Network/firewallPolicies/ruleCollectionGroups@2023-09-01' = {
  name: 'prod-rules'
  properties: {
    priority: 500
    ruleCollections: [
      {
        name: 'allow-all'
        ruleCollectionType: 'FirewallPolicyFilterRuleCollection'
        priority: 400
        action: {
          type: 'Allow'
        }
        rules: [
          {
            name: 'any-any'
            ruleType: 'NetworkRule'
            ipProtocols: [
              'Any'
            ]
            sourceAddresses: [
              '*'
            ]
            destinationAddresses: [
              '*'
            ]
            destinationPorts: [
              '*'
            ]
          }
        ]
      }
    ]
  }
}
