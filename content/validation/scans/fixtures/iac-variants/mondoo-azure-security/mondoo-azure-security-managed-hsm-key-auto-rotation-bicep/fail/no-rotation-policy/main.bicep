// A key with no rotation policy is rotated only when someone remembers to do
// it by hand, which in practice means never.
resource hsmKey 'Microsoft.KeyVault/managedHSMs/keys@2023-07-01' = {
  name: 'prod-hsm/payment-signing'
  properties: {
    kty: 'EC-HSM'
    curveName: 'P-256'
    keyOps: [
      'sign'
      'verify'
    ]
  }
}
