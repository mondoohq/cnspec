resource hsmKey 'Microsoft.KeyVault/managedHSMs/keys@2023-07-01' = {
  name: 'payment-signing'
  properties: {
    kty: 'EC-HSM'
    curveName: 'P-256'
    attributes: {
      enabled: true
      exp: 1798761600
    }
  }
}
