window.BENCHMARK_DATA = {
  "lastUpdate": 1758033302179,
  "repoUrl": "https://github.com/mondoohq/cnspec",
  "entries": {
    "Benchmark": [
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "1ba49189edf085afd3a09b06d472d9f2211e84df",
          "message": "Correct benchmark workflow to compare PRs against main",
          "timestamp": "2025-07-08T09:12:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1725/commits/1ba49189edf085afd3a09b06d472d9f2211e84df"
        },
        "date": 1751977119795,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20915,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "60811 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20915,
            "unit": "ns/op",
            "extra": "60811 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "60811 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60811 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19451,
            "unit": "ns/op\t    4900 B/op\t      71 allocs/op",
            "extra": "57562 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19451,
            "unit": "ns/op",
            "extra": "57562 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4900,
            "unit": "B/op",
            "extra": "57562 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57562 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "igor@mondoo.com",
            "name": "Igor Komlew",
            "username": "glower"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "fde18ab8fc51218eee5ea123aa038c4a390bfe4b",
          "message": "Correct benchmark workflow to compare PRs against main (#1725)",
          "timestamp": "2025-07-08T14:29:05+02:00",
          "tree_id": "8becddf618503f80a806682de5840f4be02860dd",
          "url": "https://github.com/mondoohq/cnspec/commit/fde18ab8fc51218eee5ea123aa038c4a390bfe4b"
        },
        "date": 1751977911446,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19692,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "63902 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19692,
            "unit": "ns/op",
            "extra": "63902 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "63902 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63902 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19406,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "58406 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19406,
            "unit": "ns/op",
            "extra": "58406 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "58406 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58406 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "96bcefe307c6f2acedbbf26838a5ad84bd5452a1",
          "message": "Enable runtime auto-update configuration for LocalScanner",
          "timestamp": "2025-07-08T12:29:10Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1685/commits/96bcefe307c6f2acedbbf26838a5ad84bd5452a1"
        },
        "date": 1751978017439,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 5844499835,
            "unit": "ns/op\t681353032 B/op\t15544441 allocs/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 5844499835,
            "unit": "ns/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 681353032,
            "unit": "B/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 15544441,
            "unit": "allocs/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 34508995,
            "unit": "ns/op\t15555538 B/op\t  108431 allocs/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 34508995,
            "unit": "ns/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 15555538,
            "unit": "B/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 108431,
            "unit": "allocs/op",
            "extra": "32 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "salim@afiunemaya.com.mx",
            "name": "Salim Afiune Maya",
            "username": "afiune"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "7393da96a8ab3b9b28f3a6f0cc6c2c40d5be2048",
          "message": "🐛 onboarding: ms365 does not depend on a subscription (#1722)",
          "timestamp": "2025-07-08T06:37:11-07:00",
          "tree_id": "4a90cc9550386d78afc20c8d74e8a37b94d13dce",
          "url": "https://github.com/mondoohq/cnspec/commit/7393da96a8ab3b9b28f3a6f0cc6c2c40d5be2048"
        },
        "date": 1751981864068,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 18821,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "60248 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 18821,
            "unit": "ns/op",
            "extra": "60248 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "60248 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60248 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20228,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "54308 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20228,
            "unit": "ns/op",
            "extra": "54308 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "54308 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "54308 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "2e0475111dd2fe53df12dd880066bc1084ad74de",
          "message": "🐛 onboarding: ms365 requires `Policy.Read.All` in MsGraph",
          "timestamp": "2025-07-08T13:37:16Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1723/commits/2e0475111dd2fe53df12dd880066bc1084ad74de"
        },
        "date": 1751983664141,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20719,
            "unit": "ns/op\t    4901 B/op\t      71 allocs/op",
            "extra": "67881 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20719,
            "unit": "ns/op",
            "extra": "67881 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4901,
            "unit": "B/op",
            "extra": "67881 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "67881 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 21102,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "66261 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 21102,
            "unit": "ns/op",
            "extra": "66261 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "66261 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "66261 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "salim@afiunemaya.com.mx",
            "name": "Salim Afiune Maya",
            "username": "afiune"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "2d0db013b7a56c2ff3dd331b6af38ee7e065c6ac",
          "message": "🐛 onboarding: ms365 requires `Policy.Read.All` in MsGraph (#1723)\n\nA few queries that were failing:\n\n```\nquery=h2/skzZUdl0=\nmicrosoft.conditionalAccess.policies.where(conditions.users.includeRoles.length > 0) {\n  grantControls.authenticationStrength.displayName == 'Phishing-resistant MFA'\n}\n\nquery=deTFR6B+GS4=\nmicrosoft.conditionalAccess.policies.where(conditions.applications.includeApplications.contains('Microsoft Intune Enrollment')) {\n  sessionControls.signInFrequency.frequencyInterval == 'everyTime' &&\n  sessionControls.signInFrequency.isEnabled == true &&\n  state == 'enabled'\n}\n\nquery=RFe235yOM3g=\nmicrosoft.conditionalAccess.policies.where(conditions.signInRiskLevels == 'high' || conditions.signInRiskLevels == 'medium') {\n  grantControls.builtInControls.contains('mfa')\n  sessionControls.signInFrequency.frequencyInterval == 'everyTime'\n  state == 'enabled'\n}\n\nquery=keqGtCd5aLY=\nmicrosoft.conditionalAccess.policies.where(conditions.userRiskLevels.contains('high')) {\n  grantControls.builtInControls.contains('mfa')\n  grantControls.builtInControls.contains('passwordChange')\n  sessionControls.signInFrequency.frequencyInterval == 'everyTime'\n  state == 'enabled'\n}\n```\n\n---------\n\nSigned-off-by: Salim Afiune Maya <afiune@mondoo.com>",
          "timestamp": "2025-07-08T07:14:29-07:00",
          "tree_id": "a950eb97087851a8c864ff7a6f5f1ee6cfe4ebaf",
          "url": "https://github.com/mondoohq/cnspec/commit/2d0db013b7a56c2ff3dd331b6af38ee7e065c6ac"
        },
        "date": 1751984105720,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20423,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "62812 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20423,
            "unit": "ns/op",
            "extra": "62812 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "62812 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62812 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19709,
            "unit": "ns/op\t    4899 B/op\t      71 allocs/op",
            "extra": "54590 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19709,
            "unit": "ns/op",
            "extra": "54590 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4899,
            "unit": "B/op",
            "extra": "54590 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "54590 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "3fe1b7509f5315ecc63804afc43b49aba9b9ac8d",
          "message": "🧹 Bump cnquery to v11.62.1 (#1727)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-07-08T15:52:34Z",
          "tree_id": "971a6b8f099dbf911c06b83881a5edcfacfb50f6",
          "url": "https://github.com/mondoohq/cnspec/commit/3fe1b7509f5315ecc63804afc43b49aba9b9ac8d"
        },
        "date": 1751990131344,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20973,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "59000 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20973,
            "unit": "ns/op",
            "extra": "59000 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "59000 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59000 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20683,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "59833 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20683,
            "unit": "ns/op",
            "extra": "59833 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "59833 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59833 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "91ccb4d2d9fb9303f20b85a0fb78cf5396b5a9d0",
          "message": "⭐ group reporting in CLI",
          "timestamp": "2025-07-08T15:52:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1726/commits/91ccb4d2d9fb9303f20b85a0fb78cf5396b5a9d0"
        },
        "date": 1752008165961,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 18831,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "56079 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 18831,
            "unit": "ns/op",
            "extra": "56079 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "56079 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56079 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19802,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "53580 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19802,
            "unit": "ns/op",
            "extra": "53580 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "53580 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "53580 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "b37441b988405af441722d4078accab108fc7f83",
          "message": "⭐ group reporting in CLI",
          "timestamp": "2025-07-08T15:52:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1726/commits/b37441b988405af441722d4078accab108fc7f83"
        },
        "date": 1752008414610,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20118,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "58971 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20118,
            "unit": "ns/op",
            "extra": "58971 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "58971 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58971 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20008,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "63955 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20008,
            "unit": "ns/op",
            "extra": "63955 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "63955 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63955 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "199481b4cb5ce7b736a0877c9633ccada8dc3917",
          "message": "⭐ group reporting in CLI",
          "timestamp": "2025-07-08T15:52:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1726/commits/199481b4cb5ce7b736a0877c9633ccada8dc3917"
        },
        "date": 1752008884477,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20065,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "62870 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20065,
            "unit": "ns/op",
            "extra": "62870 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "62870 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62870 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19534,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "58615 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19534,
            "unit": "ns/op",
            "extra": "58615 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "58615 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58615 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "9ea709735c3e24c14de0bd5d6e06dc9e9a0e6a71",
          "message": "⚙️  onboarding: refactor ms365 permissions",
          "timestamp": "2025-07-08T15:52:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1728/commits/9ea709735c3e24c14de0bd5d6e06dc9e9a0e6a71"
        },
        "date": 1752009418943,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20587,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "64575 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20587,
            "unit": "ns/op",
            "extra": "64575 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "64575 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64575 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20881,
            "unit": "ns/op\t    4890 B/op\t      71 allocs/op",
            "extra": "64989 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20881,
            "unit": "ns/op",
            "extra": "64989 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4890,
            "unit": "B/op",
            "extra": "64989 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64989 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "cd92de53c3428c3938485e2091a98c3aee2f794f",
          "message": "⚙️  onboarding: refactor ms365 permissions",
          "timestamp": "2025-07-08T15:52:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1728/commits/cd92de53c3428c3938485e2091a98c3aee2f794f"
        },
        "date": 1752011314930,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19741,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "65762 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19741,
            "unit": "ns/op",
            "extra": "65762 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "65762 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "65762 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 22178,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "47852 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 22178,
            "unit": "ns/op",
            "extra": "47852 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "47852 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "47852 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "953f4dbbfc5e6682c7010ecc83bd47697fc787d3",
          "message": "⚙️  onboarding: refactor ms365 permissions",
          "timestamp": "2025-07-08T15:52:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1728/commits/953f4dbbfc5e6682c7010ecc83bd47697fc787d3"
        },
        "date": 1752013430631,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20031,
            "unit": "ns/op\t    4891 B/op\t      71 allocs/op",
            "extra": "54748 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20031,
            "unit": "ns/op",
            "extra": "54748 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4891,
            "unit": "B/op",
            "extra": "54748 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "54748 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19810,
            "unit": "ns/op\t    4906 B/op\t      71 allocs/op",
            "extra": "62841 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19810,
            "unit": "ns/op",
            "extra": "62841 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4906,
            "unit": "B/op",
            "extra": "62841 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62841 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "fc6cb6ad207a0098ce87ebe9f8b4bc6cb76c7268",
          "message": "⚙️  onboarding: refactor ms365 permissions",
          "timestamp": "2025-07-08T15:52:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1728/commits/fc6cb6ad207a0098ce87ebe9f8b4bc6cb76c7268"
        },
        "date": 1752015269168,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 18769,
            "unit": "ns/op\t    4900 B/op\t      71 allocs/op",
            "extra": "64944 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 18769,
            "unit": "ns/op",
            "extra": "64944 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4900,
            "unit": "B/op",
            "extra": "64944 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64944 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19364,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "62096 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19364,
            "unit": "ns/op",
            "extra": "62096 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "62096 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62096 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "66268a79945429c60b93f918edae99aed5f13049",
          "message": "⚙️  onboarding: refactor ms365 permissions",
          "timestamp": "2025-07-08T15:52:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1728/commits/66268a79945429c60b93f918edae99aed5f13049"
        },
        "date": 1752038279843,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20476,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "63202 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20476,
            "unit": "ns/op",
            "extra": "63202 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "63202 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63202 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 22521,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "49671 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 22521,
            "unit": "ns/op",
            "extra": "49671 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "49671 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "49671 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "96bcefe307c6f2acedbbf26838a5ad84bd5452a1",
          "message": "Enable runtime auto-update configuration for LocalScanner",
          "timestamp": "2025-07-08T12:29:10Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1685/commits/96bcefe307c6f2acedbbf26838a5ad84bd5452a1"
        },
        "date": 1752048795670,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 6585743809,
            "unit": "ns/op\t681840192 B/op\t15573861 allocs/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 6585743809,
            "unit": "ns/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 681840192,
            "unit": "B/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 15573861,
            "unit": "allocs/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 35845050,
            "unit": "ns/op\t15548591 B/op\t  108430 allocs/op",
            "extra": "31 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 35845050,
            "unit": "ns/op",
            "extra": "31 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 15548591,
            "unit": "B/op",
            "extra": "31 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 108430,
            "unit": "allocs/op",
            "extra": "31 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "dominik.richter@gmail.com",
            "name": "Dominik Richter",
            "username": "arlimus"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "108d43d865643d35ffead90857e44d697e25e6c8",
          "message": "⭐ group reporting in CLI (#1726)\n\n* ⭐ group reporting in CLI\n\nThis new reporting style groups the output by failing and non-failing\nscan results. Previously you couldn't tell which result caused the\nerror-code of a scan with a score-threshold to be non-zero. Now you\nclearly see what causes it:\n\n```\n> cnspec scan [...] --score-threshold 10\n...\n\nPassing:\n✓ A check that succeeds\n\nWarning - above score threshold:\n! MEDIUM (45):     A medium check that fails\n! HIGH (20):       A high check that fails\n\nFailing - below score threshold:\n✕ CRITICAL (0):    A critical check that fails\n```\n\nEven without score-threshold, we now get much better output:\n\n```\n> cnspec scan [...]\n...\n\nPassing:\n✓ A check that succeeds\n\nFailing:\n! MEDIUM (45):     A medium check that fails\n! HIGH (20):       A high check that fails\n✕ CRITICAL (0):    A critical check that fails\n```\n\nAs you can see, failing checks are grouped together and sorted by their\nscore now.\n\nWith v12 we will further improve this by switching to risk scoring. Stay\ntuned.\n\nSigned-off-by: Dominik Richter <dominik.richter@gmail.com>\n\n* 🟢 fix tests\n\n---------\n\nSigned-off-by: Dominik Richter <dominik.richter@gmail.com>",
          "timestamp": "2025-07-09T03:56:15-07:00",
          "tree_id": "4723e34e60186ec8325582ebff7a0ab94136961d",
          "url": "https://github.com/mondoohq/cnspec/commit/108d43d865643d35ffead90857e44d697e25e6c8"
        },
        "date": 1752058609607,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19671,
            "unit": "ns/op\t    4899 B/op\t      71 allocs/op",
            "extra": "55077 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19671,
            "unit": "ns/op",
            "extra": "55077 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4899,
            "unit": "B/op",
            "extra": "55077 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "55077 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19180,
            "unit": "ns/op\t    4890 B/op\t      71 allocs/op",
            "extra": "58238 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19180,
            "unit": "ns/op",
            "extra": "58238 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4890,
            "unit": "B/op",
            "extra": "58238 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58238 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "0e47ced5996f965f1f5a41ea5538342ec3e6e0b7",
          "message": "Enable runtime auto-update configuration for LocalScanner",
          "timestamp": "2025-07-09T10:56:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1730/commits/0e47ced5996f965f1f5a41ea5538342ec3e6e0b7"
        },
        "date": 1752074838715,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 7219589853,
            "unit": "ns/op\t681824824 B/op\t15573966 allocs/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 7219589853,
            "unit": "ns/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 681824824,
            "unit": "B/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 15573966,
            "unit": "allocs/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 37678646,
            "unit": "ns/op\t15569790 B/op\t  108450 allocs/op",
            "extra": "30 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 37678646,
            "unit": "ns/op",
            "extra": "30 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 15569790,
            "unit": "B/op",
            "extra": "30 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 108450,
            "unit": "allocs/op",
            "extra": "30 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "6529c82ff3459c991a388ba29f03b15c0bda564c",
          "message": "Enable runtime auto-update configuration for LocalScanner",
          "timestamp": "2025-07-09T10:56:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1730/commits/6529c82ff3459c991a388ba29f03b15c0bda564c"
        },
        "date": 1752078155493,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19460,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "64986 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19460,
            "unit": "ns/op",
            "extra": "64986 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "64986 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64986 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20284,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "57429 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20284,
            "unit": "ns/op",
            "extra": "57429 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "57429 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57429 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "8d6973c349d7ca5e3c638477c23a3f5182f39237",
          "message": "Enable runtime auto-update configuration for LocalScanner",
          "timestamp": "2025-07-09T10:56:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1730/commits/8d6973c349d7ca5e3c638477c23a3f5182f39237"
        },
        "date": 1752078596866,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21916,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "57901 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21916,
            "unit": "ns/op",
            "extra": "57901 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "57901 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57901 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19496,
            "unit": "ns/op\t    4891 B/op\t      71 allocs/op",
            "extra": "60898 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19496,
            "unit": "ns/op",
            "extra": "60898 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4891,
            "unit": "B/op",
            "extra": "60898 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60898 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "9ee6c2093dc579f763d9d573e0b25aad44b08f9d",
          "message": "Enable runtime auto-update configuration for LocalScanner",
          "timestamp": "2025-07-09T10:56:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1730/commits/9ee6c2093dc579f763d9d573e0b25aad44b08f9d"
        },
        "date": 1752078750411,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19025,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "56380 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19025,
            "unit": "ns/op",
            "extra": "56380 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "56380 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56380 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19790,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "62599 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19790,
            "unit": "ns/op",
            "extra": "62599 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "62599 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62599 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "fba8a326afa336001688834d8fea465fa5a78422",
          "message": "Enable runtime auto-update configuration for LocalScanner",
          "timestamp": "2025-07-09T10:56:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1730/commits/fba8a326afa336001688834d8fea465fa5a78422"
        },
        "date": 1752079442325,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19140,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "72507 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19140,
            "unit": "ns/op",
            "extra": "72507 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "72507 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "72507 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19636,
            "unit": "ns/op\t    4900 B/op\t      71 allocs/op",
            "extra": "63392 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19636,
            "unit": "ns/op",
            "extra": "63392 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4900,
            "unit": "B/op",
            "extra": "63392 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63392 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "e5da5f059287b896de3288df605aebad8b56efa6",
          "message": "Enable runtime auto-update configuration for LocalScanner",
          "timestamp": "2025-07-09T10:56:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1730/commits/e5da5f059287b896de3288df605aebad8b56efa6"
        },
        "date": 1752079715594,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 5846128430,
            "unit": "ns/op\t681866920 B/op\t15573999 allocs/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 5846128430,
            "unit": "ns/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 681866920,
            "unit": "B/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 15573999,
            "unit": "allocs/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 33482723,
            "unit": "ns/op\t15558793 B/op\t  108426 allocs/op",
            "extra": "36 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 33482723,
            "unit": "ns/op",
            "extra": "36 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 15558793,
            "unit": "B/op",
            "extra": "36 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 108426,
            "unit": "allocs/op",
            "extra": "36 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "07f86f68c8175e646b23872c2d2fd1280f2e5e72",
          "message": "Enable runtime auto-update configuration for LocalScanner",
          "timestamp": "2025-07-09T10:56:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1730/commits/07f86f68c8175e646b23872c2d2fd1280f2e5e72"
        },
        "date": 1752080010010,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20015,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "61581 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20015,
            "unit": "ns/op",
            "extra": "61581 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "61581 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61581 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19874,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "64016 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19874,
            "unit": "ns/op",
            "extra": "64016 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "64016 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64016 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "eb9efcbd29e0a0a9d9aeb7fad9e7d6a70b599dfa",
          "message": "⚙️  onboarding: refactor ms365 permissions",
          "timestamp": "2025-07-09T10:56:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1728/commits/eb9efcbd29e0a0a9d9aeb7fad9e7d6a70b599dfa"
        },
        "date": 1752082487492,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20199,
            "unit": "ns/op\t    4900 B/op\t      71 allocs/op",
            "extra": "56030 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20199,
            "unit": "ns/op",
            "extra": "56030 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4900,
            "unit": "B/op",
            "extra": "56030 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56030 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19891,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "58051 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19891,
            "unit": "ns/op",
            "extra": "58051 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "58051 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58051 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "78c91fcf53ae87a5ad64f48e9f63f1db771159f8",
          "message": "⚙️  onboarding: refactor ms365 permissions",
          "timestamp": "2025-07-09T10:56:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1728/commits/78c91fcf53ae87a5ad64f48e9f63f1db771159f8"
        },
        "date": 1752089151567,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19183,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "57302 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19183,
            "unit": "ns/op",
            "extra": "57302 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "57302 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57302 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19485,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "56088 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19485,
            "unit": "ns/op",
            "extra": "56088 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "56088 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56088 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "eb11dca223c6e45493ef2d2bc63346a11c303bd3",
          "message": "⚙️  onboarding: refactor ms365 permissions",
          "timestamp": "2025-07-09T10:56:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1728/commits/eb11dca223c6e45493ef2d2bc63346a11c303bd3"
        },
        "date": 1752089221108,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19261,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "61965 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19261,
            "unit": "ns/op",
            "extra": "61965 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "61965 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61965 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 22863,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "61214 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 22863,
            "unit": "ns/op",
            "extra": "61214 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "61214 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61214 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "3239fba594df7bed9a842603dc8e35032572bf9c",
          "message": "⚙️  onboarding: refactor ms365 permissions",
          "timestamp": "2025-07-09T10:56:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1728/commits/3239fba594df7bed9a842603dc8e35032572bf9c"
        },
        "date": 1752089285850,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19228,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "60853 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19228,
            "unit": "ns/op",
            "extra": "60853 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "60853 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60853 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20123,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "69172 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20123,
            "unit": "ns/op",
            "extra": "69172 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "69172 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "69172 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "56203fc08df8b566ed52447c671bdebad06789c3",
          "message": "⚙️  onboarding: refactor ms365 permissions",
          "timestamp": "2025-07-09T10:56:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1728/commits/56203fc08df8b566ed52447c671bdebad06789c3"
        },
        "date": 1752095785395,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20404,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "62272 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20404,
            "unit": "ns/op",
            "extra": "62272 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "62272 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62272 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20508,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "66460 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20508,
            "unit": "ns/op",
            "extra": "66460 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "66460 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "66460 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "igor@mondoo.com",
            "name": "Igor Komlew",
            "username": "glower"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "b25175096e69ce054727bd694c8bd4a7853f5afd",
          "message": "Enable runtime auto-update configuration for LocalScanner (#1730)",
          "timestamp": "2025-07-10T08:15:06+02:00",
          "tree_id": "0fa9d8471a171a3fa358434505d53550f1db7a14",
          "url": "https://github.com/mondoohq/cnspec/commit/b25175096e69ce054727bd694c8bd4a7853f5afd"
        },
        "date": 1752128149231,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19761,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "69930 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19761,
            "unit": "ns/op",
            "extra": "69930 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "69930 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "69930 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19531,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "66879 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19531,
            "unit": "ns/op",
            "extra": "66879 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "66879 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "66879 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "salim@afiunemaya.com.mx",
            "name": "Salim Afiune Maya",
            "username": "afiune"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "b5620922c53b260f4c2659eac8d766bbbe30d70d",
          "message": "⚙️  onboarding: refactor ms365 permissions (#1728)\n\n* ⚙️  onboarding: refactor ms365 permissions\n\nSigned-off-by: Salim Afiune Maya <afiune@mondoo.com>\n\n* 🔐 onboarding: ms365 permission `SecurityEvents.Read.All`\n\nSigned-off-by: Salim Afiune Maya <afiune@mondoo.com>\n\n* 🔐 onboarding: ms365 SharePoint permission `Sites.FullControl.All`\n\nThe resource implementation uses three PnP PowerShell commands:\n```\nGet-PnPTenant\n\nGet-PnPTenantSyncClientRestriction\n\nGet-PnPTenantSite\n```\n\nSigned-off-by: Salim Afiune Maya <afiune@mondoo.com>\n\n* 🔐 onboarding: ms365 permission `OrgSettings-Forms.Read.All`\n\nSigned-off-by: Salim Afiune Maya <afiune@mondoo.com>\n\n* 🔐 onboarding: ms365 ExchangeOnline permission `Exchange.ManageAsApp`\n\nSigned-off-by: Salim Afiune Maya <afiune@mondoo.com>\n\n* 🔐 onboarding: ms365 permission `DeviceManagementConfiguration.Read.All`\n\nSigned-off-by: Salim Afiune Maya <afiune@mondoo.com>\n\n* 🔐 onboarding: ms365 more `DeviceManagement` permissions\n\nThe resources that need these permissions are:\n```\nmicrosoft.devicemanagement.deviceEnrollmentConfigurations\nmicrosoft.devicemanagement.managedDevices\n```\n\nSigned-off-by: Salim Afiune Maya <afiune@mondoo.com>\n\n* 🔐 onboarding: ms365 permission `OrgSettings-AppsAndServices.Read.All`\n\nFixes `microsoft.tenant.settings` introduced with https://github.com/mondoohq/cnquery/pull/5655\n\nSigned-off-by: Salim Afiune Maya <afiune@mondoo.com>\n\n---------\n\nSigned-off-by: Salim Afiune Maya <afiune@mondoo.com>",
          "timestamp": "2025-07-10T17:14:12-06:00",
          "tree_id": "ece3b8a5bd306e28a3ce9aca504894332e36099e",
          "url": "https://github.com/mondoohq/cnspec/commit/b5620922c53b260f4c2659eac8d766bbbe30d70d"
        },
        "date": 1752189298510,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21803,
            "unit": "ns/op\t    4890 B/op\t      71 allocs/op",
            "extra": "55766 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21803,
            "unit": "ns/op",
            "extra": "55766 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4890,
            "unit": "B/op",
            "extra": "55766 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "55766 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 21037,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "55831 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 21037,
            "unit": "ns/op",
            "extra": "55831 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "55831 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "55831 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "71fc175d9405644b5308e1cdc33abb8a5885e7b1",
          "message": "Bump the gomodupdates group with 4 updates",
          "timestamp": "2025-07-11T10:05:43Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1732/commits/71fc175d9405644b5308e1cdc33abb8a5885e7b1"
        },
        "date": 1752483453003,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21481,
            "unit": "ns/op\t    4890 B/op\t      71 allocs/op",
            "extra": "60572 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21481,
            "unit": "ns/op",
            "extra": "60572 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4890,
            "unit": "B/op",
            "extra": "60572 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60572 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 22159,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "51105 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 22159,
            "unit": "ns/op",
            "extra": "51105 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "51105 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "51105 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "50b673245006ce2457584862fd216a270edf037b",
          "message": "🧹 Bump cnquery to v11.63.0 (#1733)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-07-15T11:40:54Z",
          "tree_id": "f0b2d320c0b075006479783b802e9d8f7895a130",
          "url": "https://github.com/mondoohq/cnspec/commit/50b673245006ce2457584862fd216a270edf037b"
        },
        "date": 1752579820599,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19047,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "57091 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19047,
            "unit": "ns/op",
            "extra": "57091 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "57091 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57091 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19228,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "64249 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19228,
            "unit": "ns/op",
            "extra": "64249 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "64249 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64249 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "6ed41782b473ea597e49b87c37789aa561d41a04",
          "message": "Prep work to support v2 policies",
          "timestamp": "2025-07-15T11:40:58Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1734/commits/6ed41782b473ea597e49b87c37789aa561d41a04"
        },
        "date": 1752687834663,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21000,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "57328 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21000,
            "unit": "ns/op",
            "extra": "57328 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "57328 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57328 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 22134,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "52365 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 22134,
            "unit": "ns/op",
            "extra": "52365 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "52365 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "52365 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "afae7f20cf05e361ed2d8a4b25f3a72dea0c086e",
          "message": "🧹 Bump cnquery to v11.63.1 (#1735)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-07-21T15:05:33Z",
          "tree_id": "cbd5315d6f39b13525456c027100de10f40fd0e2",
          "url": "https://github.com/mondoohq/cnspec/commit/afae7f20cf05e361ed2d8a4b25f3a72dea0c086e"
        },
        "date": 1753110507258,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19643,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "59734 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19643,
            "unit": "ns/op",
            "extra": "59734 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "59734 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59734 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19316,
            "unit": "ns/op\t    4891 B/op\t      71 allocs/op",
            "extra": "63373 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19316,
            "unit": "ns/op",
            "extra": "63373 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4891,
            "unit": "B/op",
            "extra": "63373 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63373 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "e06b6ec49e3c78241b764c63eb00d38bf92d4d7b",
          "message": "🧹 Bump cnquery to v11.64.0 (#1736)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-07-22T11:59:15Z",
          "tree_id": "a5f2fdbbb3f555df3914b8471f6c043ae78a2543",
          "url": "https://github.com/mondoohq/cnspec/commit/e06b6ec49e3c78241b764c63eb00d38bf92d4d7b"
        },
        "date": 1753185720788,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21378,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "66792 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21378,
            "unit": "ns/op",
            "extra": "66792 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "66792 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "66792 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19236,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "58582 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19236,
            "unit": "ns/op",
            "extra": "58582 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "58582 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58582 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "046548b446c88d940050796c5228d1e94aca8ace",
          "message": "Bump the gomodupdates group with 7 updates",
          "timestamp": "2025-07-26T02:23:57Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1746/commits/046548b446c88d940050796c5228d1e94aca8ace"
        },
        "date": 1753717201548,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19141,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "58470 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19141,
            "unit": "ns/op",
            "extra": "58470 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "58470 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58470 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19969,
            "unit": "ns/op\t    4899 B/op\t      71 allocs/op",
            "extra": "58132 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19969,
            "unit": "ns/op",
            "extra": "58132 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4899,
            "unit": "B/op",
            "extra": "58132 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58132 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "3336fd171a8e866c1bc02b8028ba878525690674",
          "message": "Bump github.com/olekukonko/tablewriter from 0.0.5 to 1.0.9",
          "timestamp": "2025-07-26T02:23:57Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1747/commits/3336fd171a8e866c1bc02b8028ba878525690674"
        },
        "date": 1753717211954,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19956,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "58899 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19956,
            "unit": "ns/op",
            "extra": "58899 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "58899 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58899 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19916,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "59606 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19916,
            "unit": "ns/op",
            "extra": "59606 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "59606 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59606 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "63d3da6d58301bd791f5b6548364b134f1c9b210",
          "message": "🧹 Bump cnquery to v11.65.0 (#1749)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-07-29T21:08:26Z",
          "tree_id": "3f43a3cddeb732dac2e3977509e3b925bed2e5ae",
          "url": "https://github.com/mondoohq/cnspec/commit/63d3da6d58301bd791f5b6548364b134f1c9b210"
        },
        "date": 1753823473052,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20166,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "64335 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20166,
            "unit": "ns/op",
            "extra": "64335 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "64335 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64335 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19422,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "57543 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19422,
            "unit": "ns/op",
            "extra": "57543 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "57543 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57543 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "a5df6e250e5019dd58278bd493b72730b6f13cc6",
          "message": "Make properties work properly",
          "timestamp": "2025-07-30T01:52:00Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1734/commits/a5df6e250e5019dd58278bd493b72730b6f13cc6"
        },
        "date": 1753874028718,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21372,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "68560 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21372,
            "unit": "ns/op",
            "extra": "68560 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "68560 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "68560 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19463,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "55442 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19463,
            "unit": "ns/op",
            "extra": "55442 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "55442 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "55442 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "a51a00de868171d4dbadc8088269725eb7358e0c",
          "message": "Make properties work properly",
          "timestamp": "2025-07-30T18:24:29Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1734/commits/a51a00de868171d4dbadc8088269725eb7358e0c"
        },
        "date": 1753906121821,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19623,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "58312 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19623,
            "unit": "ns/op",
            "extra": "58312 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "58312 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58312 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20206,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "63580 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20206,
            "unit": "ns/op",
            "extra": "63580 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "63580 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63580 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "d717741fed2938a168287714c19e478e1c5e5891",
          "message": "✨ Fail linting if query has no datapoints/entrypoints.",
          "timestamp": "2025-07-30T18:24:29Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1755/commits/d717741fed2938a168287714c19e478e1c5e5891"
        },
        "date": 1753966025063,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19038,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "66818 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19038,
            "unit": "ns/op",
            "extra": "66818 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "66818 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "66818 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 21010,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "59673 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 21010,
            "unit": "ns/op",
            "extra": "59673 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "59673 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59673 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "c6654939c8f7568e0722db26f7914ed9394913bd",
          "message": "🐛 Fix regex for policy/query uid validation.",
          "timestamp": "2025-07-30T18:24:29Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1756/commits/c6654939c8f7568e0722db26f7914ed9394913bd"
        },
        "date": 1753968652195,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21519,
            "unit": "ns/op\t    4900 B/op\t      71 allocs/op",
            "extra": "61207 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21519,
            "unit": "ns/op",
            "extra": "61207 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4900,
            "unit": "B/op",
            "extra": "61207 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61207 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 21083,
            "unit": "ns/op\t    4891 B/op\t      71 allocs/op",
            "extra": "61957 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 21083,
            "unit": "ns/op",
            "extra": "61957 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4891,
            "unit": "B/op",
            "extra": "61957 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61957 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "2e28326824e47a9a6a9c32028368a52b2f9689a5",
          "message": "🐛 Fix regex for policy/query uid validation.",
          "timestamp": "2025-07-30T18:24:29Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1756/commits/2e28326824e47a9a6a9c32028368a52b2f9689a5"
        },
        "date": 1753970353563,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19313,
            "unit": "ns/op\t    4900 B/op\t      71 allocs/op",
            "extra": "66584 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19313,
            "unit": "ns/op",
            "extra": "66584 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4900,
            "unit": "B/op",
            "extra": "66584 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "66584 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20039,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "60692 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20039,
            "unit": "ns/op",
            "extra": "60692 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "60692 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60692 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "33118944972500ecce876431b9ed5098a52afc77",
          "message": "Make properties work properly",
          "timestamp": "2025-07-30T18:24:29Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1734/commits/33118944972500ecce876431b9ed5098a52afc77"
        },
        "date": 1753976458907,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19691,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "63889 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19691,
            "unit": "ns/op",
            "extra": "63889 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "63889 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63889 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20153,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "66476 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20153,
            "unit": "ns/op",
            "extra": "66476 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "66476 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "66476 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "preslavgerchevmail@gmail.com",
            "name": "Preslav Gerchev",
            "username": "preslavgerchev"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "cd2eb23f59e88659f028a03ab981a1e1a5d3c3d0",
          "message": "✨ Fail linting if query has no datapoints/entrypoints. (#1755)\n\nSigned-off-by: Preslav <preslav@mondoo.com>",
          "timestamp": "2025-07-31T19:00:27+03:00",
          "tree_id": "ede3270171f9738399e152b06875aceb37b4998d",
          "url": "https://github.com/mondoohq/cnspec/commit/cd2eb23f59e88659f028a03ab981a1e1a5d3c3d0"
        },
        "date": 1753977665207,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21188,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "58090 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21188,
            "unit": "ns/op",
            "extra": "58090 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "58090 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58090 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 21251,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "56842 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 21251,
            "unit": "ns/op",
            "extra": "56842 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "56842 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56842 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "0b03bb75e665d6593245936f8cc17d0a524331d2",
          "message": "Make properties work properly",
          "timestamp": "2025-07-31T16:00:31Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1734/commits/0b03bb75e665d6593245936f8cc17d0a524331d2"
        },
        "date": 1753984114205,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 18576,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "58807 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 18576,
            "unit": "ns/op",
            "extra": "58807 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "58807 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58807 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20324,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "62354 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20324,
            "unit": "ns/op",
            "extra": "62354 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "62354 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62354 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "jay@mondoo.com",
            "name": "Jay Mundrawala",
            "username": "jaym"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "72359ac844a854ca0e263e42886f498c28ba4da7",
          "message": "Make properties work properly (#1734)\n\n* Make properties work properly\n\n- Props are now namespaced to either a policy or query\n- Props on policies need to reference props through `for`\n- We will automatically make the `for` references if a pack does\n  not define any props but includes queries with props\n\nExamples:\n```yaml\npolicies:\n  - uid: example1\n    name: Example policy 1\n    version: \"1.0.0\"\n    authors:\n      - name: Mondoo\n        email: hello@mondoo.com\n    groups:\n      - title: group1\n        filters: return true\n        queries:\n          - uid: variant-1\n          - uid: variant-2\n          - uid: variant-3\n          - uid: variant-4\n    props:\n      - uid: userHome\n        for:\n          - uid: home\n          - uid: homeDir\n        mql: return \"ex\"\n\nqueries:\n  - uid: variant-1\n    mql: props.home + \" on 1\"\n    props:\n      - uid: home\n        mql: return \"p1\"\n\n  - uid: variant-2\n    mql: props.home + \" on 2\"\n    props:\n      - uid: home\n        mql: return \"p2\"\n\n  - uid: variant-3\n    mql: props.homeDir + \" on 3\"\n    props:\n      - uid: homeDir\n        mql: return \"p3\"\n\n  - uid: variant-4\n    mql: props.user + \" is the user\"\n    props:\n      - uid: user\n        mql: return \"ada\"\n```\n\n```\ngo run ./apps/cnspec scan -f props.mql.yaml\n```\n\n```\nhome.+: \"ex on 1\"\nhome.+: \"ex on 2\"\nhomeDir.+: \"ex on 3\"\nuser.+: \"ada is the user\"\n```\n\n```\ngo run ./apps/cnspec scan -f props.mql.yaml --props userHome=\"return 'foo'\"\n```\n\n```\nhomeDir.+: \"foo on 3\"\nuser.+: \"ada is the user\"\nhome.+: \"foo on 1\"\nhome.+: \"foo on 2\"\n```\n\n```yaml\npolicies:\n  - uid: example1\n    name: Example policy 1\n    version: \"1.0.0\"\n    authors:\n      - name: Mondoo\n        email: hello@mondoo.com\n    groups:\n      - title: group1\n        filters: return true\n        queries:\n          - uid: variant-1\n          - uid: variant-2\n          - uid: variant-3\n          - uid: variant-4\nqueries:\n  - uid: variant-1\n    mql: props.home + \" on 1\"\n    props:\n      - uid: home\n        mql: return \"p1\"\n\n  - uid: variant-2\n    mql: props.home + \" on 2\"\n    props:\n      - uid: home\n        mql: return \"p2\"\n\n  - uid: variant-3\n    mql: props.homeDir + \" on 3\"\n    props:\n      - uid: homeDir\n        mql: return \"p3\"\n\n  - uid: variant-4\n    mql: props.user + \" is the user\"\n    props:\n      - uid: user\n        mql: return \"ada\"\n```\n\n```\ngo run ./apps/cnspec scan -f props.mql.yaml\n```\n\n```\nhome.+: \"p2 on 2\"\nhome.+: \"p1 on 1\"\nhomeDir.+: \"p3 on 3\"\nuser.+: \"ada is the user\"\n```\n\n```\ngo run ./apps/cnspec scan -f props.mql.yaml --props home=\"return 'foo'\"\n```\n\n```\nhome.+: \"foo on 2\"\nhomeDir.+: \"p3 on 3\"\nuser.+: \"ada is the user\"\nhome.+: \"foo on 1\"\n```\n\n* try to lift properties early",
          "timestamp": "2025-07-31T13:05:26-05:00",
          "tree_id": "766c7d974d010eb074cef1259ca45c76e55bf1ef",
          "url": "https://github.com/mondoohq/cnspec/commit/72359ac844a854ca0e263e42886f498c28ba4da7"
        },
        "date": 1753985298441,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19769,
            "unit": "ns/op\t    4891 B/op\t      71 allocs/op",
            "extra": "55077 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19769,
            "unit": "ns/op",
            "extra": "55077 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4891,
            "unit": "B/op",
            "extra": "55077 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "55077 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20489,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "65078 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20489,
            "unit": "ns/op",
            "extra": "65078 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "65078 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "65078 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "e813c49fe75874e477053310c993e8118e078fd7",
          "message": "🧹 Migrate deprecated Query field on Mquery",
          "timestamp": "2025-08-01T06:21:37Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1757/commits/e813c49fe75874e477053310c993e8118e078fd7"
        },
        "date": 1754062769754,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19879,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "64268 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19879,
            "unit": "ns/op",
            "extra": "64268 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "64268 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64268 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20162,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "63889 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20162,
            "unit": "ns/op",
            "extra": "63889 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "63889 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63889 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "jay@mondoo.com",
            "name": "Jay Mundrawala",
            "username": "jaym"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "efa48552706eb229e20ca73ef2df591a8c0bbcee",
          "message": "🧹 Migrate deprecated Query field on Mquery (#1757)\n\nThe order in which the bundle is compiled has changed, which meant the\nold field was not properly being migrated",
          "timestamp": "2025-08-01T10:50:26-05:00",
          "tree_id": "dd01e1f78fd6415a573e0b311980d638c0eb6a05",
          "url": "https://github.com/mondoohq/cnspec/commit/efa48552706eb229e20ca73ef2df591a8c0bbcee"
        },
        "date": 1754063459904,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20175,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "57087 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20175,
            "unit": "ns/op",
            "extra": "57087 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "57087 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57087 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 21072,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "52915 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 21072,
            "unit": "ns/op",
            "extra": "52915 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "52915 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "52915 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "fa1bd3432122704b08ff2ef41db7aac849d4f9b4",
          "message": "Bump the gomodupdates group with 3 updates",
          "timestamp": "2025-08-02T01:36:10Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1762/commits/fa1bd3432122704b08ff2ef41db7aac849d4f9b4"
        },
        "date": 1754305280226,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19030,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "71568 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19030,
            "unit": "ns/op",
            "extra": "71568 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "71568 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "71568 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20163,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "62470 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20163,
            "unit": "ns/op",
            "extra": "62470 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "62470 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62470 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "ea03d80358a91b463f34361e9c2f70464410c53b",
          "message": "🧹 Bump cnquery to v11.66.0 (#1767)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-08-05T11:14:24Z",
          "tree_id": "d59fe15037db8ef6a1be28c46af974948ab5a22b",
          "url": "https://github.com/mondoohq/cnspec/commit/ea03d80358a91b463f34361e9c2f70464410c53b"
        },
        "date": 1754392630885,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19776,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "55515 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19776,
            "unit": "ns/op",
            "extra": "55515 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "55515 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "55515 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 21293,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "64969 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 21293,
            "unit": "ns/op",
            "extra": "64969 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "64969 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64969 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "83eee2f2a966bd90d25e7a2f52f63e4af12e7b49",
          "message": "🧹 Bump cnquery to v11.66.1 (#1769)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-08-06T12:13:27Z",
          "tree_id": "8816770369a6ecada0a34726ac3d1a08cb44edd9",
          "url": "https://github.com/mondoohq/cnspec/commit/83eee2f2a966bd90d25e7a2f52f63e4af12e7b49"
        },
        "date": 1754482585357,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19909,
            "unit": "ns/op\t    4890 B/op\t      71 allocs/op",
            "extra": "59143 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19909,
            "unit": "ns/op",
            "extra": "59143 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4890,
            "unit": "B/op",
            "extra": "59143 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59143 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20494,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "58353 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20494,
            "unit": "ns/op",
            "extra": "58353 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "58353 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58353 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "da69516e4db3ca99821e471417048ba4a1e6469d",
          "message": "Bump the gomodupdates group with 7 updates",
          "timestamp": "2025-08-10T02:10:52Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1775/commits/da69516e4db3ca99821e471417048ba4a1e6469d"
        },
        "date": 1754916214993,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20095,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "64946 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20095,
            "unit": "ns/op",
            "extra": "64946 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "64946 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64946 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19970,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "70018 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19970,
            "unit": "ns/op",
            "extra": "70018 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "70018 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "70018 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "852b47172de2104422b0992629db009f393c6109",
          "message": "⭐ valid until",
          "timestamp": "2025-08-10T02:10:52Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1754/commits/852b47172de2104422b0992629db009f393c6109"
        },
        "date": 1754921487506,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19677,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "58414 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19677,
            "unit": "ns/op",
            "extra": "58414 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "58414 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58414 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19813,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "60420 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19813,
            "unit": "ns/op",
            "extra": "60420 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "60420 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60420 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "dominik.richter@gmail.com",
            "name": "Dominik Richter",
            "username": "arlimus"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "cbbcdc0de62701a48f13f7571f847a552bbe749d",
          "message": "⭐ valid until (#1754)\n\n* ⭐ valid until\n\nIntroduces the `valid` keyword in policies, which supports setting an\n`until` value. This allows us to create human-readable policy groups\nthat are configured for a limited time.\n\nThis is particularly useful when defining temporary exceptions:\n\n```\npolicies:\n  - uid: example1\n    name: Example policy 1\n    groups:\n      - filters:\n          - mql: asset.family.contains('unix')\n        checks:\n          - uid: check-05\n            title: SSHd should only use very secure ciphers\n            mql: |\n              sshd.config.ciphers.all( _ == /ctr/ )\n            impact: 95\n\n      - type: override\n        title: Exception for strong ciphers until September\n        valid:\n          until: 2025-09-01\n        checks:\n          - uid: check-05\n            action: preview\n```\n\nDepends on https://github.com/mondoohq/cnquery/pull/5817\n\n* 🧹 fix genai mistakes\n\nSigned-off-by: Dominik Richter <dominik.richter@gmail.com>\n\n* 🧹 linter suggestion\n\n* jays changes\n\n* update recalculateAt\n\n* update cnquery\n\n* update policy checksums\n\n* fix tests\n\n---------\n\nSigned-off-by: Dominik Richter <dominik.richter@gmail.com>\nCo-authored-by: Jay Mundrawala <jay@mondoo.com>",
          "timestamp": "2025-08-11T09:17:35-05:00",
          "tree_id": "4f7feca6f0033207428671fcdc8f9367ff471684",
          "url": "https://github.com/mondoohq/cnspec/commit/cbbcdc0de62701a48f13f7571f847a552bbe749d"
        },
        "date": 1754922028417,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19760,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "61632 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19760,
            "unit": "ns/op",
            "extra": "61632 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "61632 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61632 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 22297,
            "unit": "ns/op\t    4891 B/op\t      71 allocs/op",
            "extra": "60628 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 22297,
            "unit": "ns/op",
            "extra": "60628 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4891,
            "unit": "B/op",
            "extra": "60628 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60628 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "d4b3c2900636e21f9e0df8db95734dda3545a325",
          "message": "Bump github.com/olekukonko/tablewriter from 0.0.5 to 1.0.9",
          "timestamp": "2025-08-11T14:17:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1747/commits/d4b3c2900636e21f9e0df8db95734dda3545a325"
        },
        "date": 1754922133518,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19968,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "53259 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19968,
            "unit": "ns/op",
            "extra": "53259 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "53259 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "53259 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20827,
            "unit": "ns/op\t    4890 B/op\t      71 allocs/op",
            "extra": "59666 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20827,
            "unit": "ns/op",
            "extra": "59666 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4890,
            "unit": "B/op",
            "extra": "59666 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59666 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "7095a958daebc34419c39b7353f72facb50b7796",
          "message": "🐛 Don't change score type",
          "timestamp": "2025-08-11T14:17:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1776/commits/7095a958daebc34419c39b7353f72facb50b7796"
        },
        "date": 1754934539222,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19211,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "60650 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19211,
            "unit": "ns/op",
            "extra": "60650 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "60650 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60650 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20320,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "60811 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20320,
            "unit": "ns/op",
            "extra": "60811 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "60811 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60811 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "jay@mondoo.com",
            "name": "Jay Mundrawala",
            "username": "jaym"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "88b2c82b7ec6da4996d1b3c350270ab5757ed159",
          "message": "🐛 Don't change score type (#1776)\n\nThis turns out to be a breaking change. There's also no need to really\ndo this. Its not going to be counted against any policy and it looked\nlike this was only done for printing reasons\n\nBroke in #1754",
          "timestamp": "2025-08-11T13:22:05-05:00",
          "tree_id": "c17ab3b46410068315b1c96f099322c1d84cf7b0",
          "url": "https://github.com/mondoohq/cnspec/commit/88b2c82b7ec6da4996d1b3c350270ab5757ed159"
        },
        "date": 1754936560569,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19446,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "58149 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19446,
            "unit": "ns/op",
            "extra": "58149 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "58149 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58149 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20134,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "67304 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20134,
            "unit": "ns/op",
            "extra": "67304 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "67304 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "67304 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "3215eeafc3deba283bfcfc40a43c020cd92304c2",
          "message": "Bump the gomodupdates group across 1 directory with 6 updates",
          "timestamp": "2025-08-11T18:22:09Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1777/commits/3215eeafc3deba283bfcfc40a43c020cd92304c2"
        },
        "date": 1754940031522,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20446,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "58156 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20446,
            "unit": "ns/op",
            "extra": "58156 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "58156 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58156 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20617,
            "unit": "ns/op\t    4889 B/op\t      71 allocs/op",
            "extra": "53445 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20617,
            "unit": "ns/op",
            "extra": "53445 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4889,
            "unit": "B/op",
            "extra": "53445 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "53445 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "c10a34b96c3d8313c67f376171d6502240966796",
          "message": "🧹 Bump cnquery to v11.67.0 (#1778)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-08-12T10:21:11Z",
          "tree_id": "f2647081966a50c6c5e02f395f41a80cfb6a875a",
          "url": "https://github.com/mondoohq/cnspec/commit/c10a34b96c3d8313c67f376171d6502240966796"
        },
        "date": 1754994244233,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21072,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "61465 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21072,
            "unit": "ns/op",
            "extra": "61465 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "61465 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61465 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19787,
            "unit": "ns/op\t    4899 B/op\t      71 allocs/op",
            "extra": "60774 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19787,
            "unit": "ns/op",
            "extra": "60774 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4899,
            "unit": "B/op",
            "extra": "60774 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60774 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "98cbcf819c1ece27e1a8106a9110c379e394b363",
          "message": "🐛 Fix MaxInt type",
          "timestamp": "2025-08-12T10:21:15Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1779/commits/98cbcf819c1ece27e1a8106a9110c379e394b363"
        },
        "date": 1754999161830,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20424,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "58098 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20424,
            "unit": "ns/op",
            "extra": "58098 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "58098 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58098 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20206,
            "unit": "ns/op\t    4891 B/op\t      71 allocs/op",
            "extra": "58928 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20206,
            "unit": "ns/op",
            "extra": "58928 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4891,
            "unit": "B/op",
            "extra": "58928 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58928 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "827818+czunker@users.noreply.github.com",
            "name": "Christian Zunker",
            "username": "czunker"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "5cce7fd8441d68d74286ad52251db5345290821a",
          "message": "🐛 Fix MaxInt type (#1779)\n\nThat should fix:\nhttps://github.com/mondoohq/cnspec/actions/runs/16906004500/job/47895837435\n\nSigned-off-by: Christian Zunker <christian@mondoo.com>",
          "timestamp": "2025-08-12T13:52:31+02:00",
          "tree_id": "1dbae5b3de90451c432d8ca626411a9c033a6b3b",
          "url": "https://github.com/mondoohq/cnspec/commit/5cce7fd8441d68d74286ad52251db5345290821a"
        },
        "date": 1754999582942,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19258,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "57462 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19258,
            "unit": "ns/op",
            "extra": "57462 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "57462 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57462 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19053,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "53900 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19053,
            "unit": "ns/op",
            "extra": "53900 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "53900 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "53900 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "a6340e2520a1e21435a4d36ee7498b1ab9e81020",
          "message": "🧹 Bump cnquery to v11.67.1 (#1780)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-08-12T13:04:53Z",
          "tree_id": "1359c69a00919a03ae1b1d5c234398682e69e6c3",
          "url": "https://github.com/mondoohq/cnspec/commit/a6340e2520a1e21435a4d36ee7498b1ab9e81020"
        },
        "date": 1755004065371,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19537,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "61708 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19537,
            "unit": "ns/op",
            "extra": "61708 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "61708 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61708 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 21044,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "50690 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 21044,
            "unit": "ns/op",
            "extra": "50690 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "50690 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "50690 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "d0bb31945f4d2e29bf1f57198b853aaba9c30c1f",
          "message": "⚙️ Use cnquery's build flags to disable the max message size limitation.",
          "timestamp": "2025-08-12T22:01:44Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1782/commits/d0bb31945f4d2e29bf1f57198b853aaba9c30c1f"
        },
        "date": 1755093412904,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19797,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "54986 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19797,
            "unit": "ns/op",
            "extra": "54986 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "54986 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "54986 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20044,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "59784 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20044,
            "unit": "ns/op",
            "extra": "59784 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "59784 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59784 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "preslavgerchevmail@gmail.com",
            "name": "Preslav Gerchev",
            "username": "preslavgerchev"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "5debf9eed4980903b5d11cb01ec425d81cf69d99",
          "message": "⚙️ Use cnquery's build flags to disable the max message size limitation. (#1782)\n\nSigned-off-by: Preslav <preslav@mondoo.com>",
          "timestamp": "2025-08-13T16:58:06+03:00",
          "tree_id": "f1f75c39ab2dac19c074d9ec9b82d92fb737b578",
          "url": "https://github.com/mondoohq/cnspec/commit/5debf9eed4980903b5d11cb01ec425d81cf69d99"
        },
        "date": 1755093650155,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20795,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "51988 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20795,
            "unit": "ns/op",
            "extra": "51988 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "51988 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "51988 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20384,
            "unit": "ns/op\t    4899 B/op\t      71 allocs/op",
            "extra": "56596 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20384,
            "unit": "ns/op",
            "extra": "56596 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4899,
            "unit": "B/op",
            "extra": "56596 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56596 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "35d5143a762c82aaba2ce8d703e037346e487180",
          "message": "Bump actions/checkout from 4 to 5",
          "timestamp": "2025-08-15T07:32:56Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1790/commits/35d5143a762c82aaba2ce8d703e037346e487180"
        },
        "date": 1755511726964,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20448,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "62797 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20448,
            "unit": "ns/op",
            "extra": "62797 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "62797 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62797 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20716,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "54018 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20716,
            "unit": "ns/op",
            "extra": "54018 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "54018 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "54018 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "69fb05ec36448691b3aaec7ffb20e6f8f4abc746",
          "message": "Bump the gomodupdates group with 3 updates",
          "timestamp": "2025-08-15T07:32:56Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1791/commits/69fb05ec36448691b3aaec7ffb20e6f8f4abc746"
        },
        "date": 1755512048479,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21938,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "54051 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21938,
            "unit": "ns/op",
            "extra": "54051 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "54051 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "54051 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20400,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "60542 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20400,
            "unit": "ns/op",
            "extra": "60542 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "60542 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60542 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "9a72de6f108c8bcfed888122437b4020a598585b",
          "message": "🧹 Bump cnquery to v11.68.0 (#1796)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-08-19T08:37:56Z",
          "tree_id": "5166ffdc2564c01a3670b4bef72ef2788bfb0e51",
          "url": "https://github.com/mondoohq/cnspec/commit/9a72de6f108c8bcfed888122437b4020a598585b"
        },
        "date": 1755592833959,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20725,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "60757 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20725,
            "unit": "ns/op",
            "extra": "60757 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "60757 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60757 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 21374,
            "unit": "ns/op\t    4889 B/op\t      71 allocs/op",
            "extra": "59912 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 21374,
            "unit": "ns/op",
            "extra": "59912 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4889,
            "unit": "B/op",
            "extra": "59912 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59912 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "14c58e3a14ec72a1758e9fc28948b52a49e9e556",
          "message": "Bump the gomodupdates group across 1 directory with 6 updates",
          "timestamp": "2025-08-21T01:43:05Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1802/commits/14c58e3a14ec72a1758e9fc28948b52a49e9e556"
        },
        "date": 1756127450476,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 22064,
            "unit": "ns/op\t    4889 B/op\t      71 allocs/op",
            "extra": "62596 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 22064,
            "unit": "ns/op",
            "extra": "62596 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4889,
            "unit": "B/op",
            "extra": "62596 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62596 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 24091,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "49280 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 24091,
            "unit": "ns/op",
            "extra": "49280 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "49280 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "49280 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "fe330e75cbabed51cee6b6a1d2cc817673287aac",
          "message": "⭐ policy require providers",
          "timestamp": "2025-08-25T13:13:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1803/commits/fe330e75cbabed51cee6b6a1d2cc817673287aac"
        },
        "date": 1756192562562,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 18796,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "65905 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 18796,
            "unit": "ns/op",
            "extra": "65905 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "65905 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "65905 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20012,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "59194 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20012,
            "unit": "ns/op",
            "extra": "59194 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "59194 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59194 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "66b649bf321fbae44861b63215b7fec1327c37ea",
          "message": "⭐ policy require providers",
          "timestamp": "2025-08-25T13:13:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1803/commits/66b649bf321fbae44861b63215b7fec1327c37ea"
        },
        "date": 1756192598111,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19450,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "56204 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19450,
            "unit": "ns/op",
            "extra": "56204 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "56204 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56204 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19414,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "60319 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19414,
            "unit": "ns/op",
            "extra": "60319 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "60319 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60319 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "06568f9eb2a762f58f19b6cca6a99d35ab3b90ad",
          "message": "⭐ policy require providers",
          "timestamp": "2025-08-25T13:13:19Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1803/commits/06568f9eb2a762f58f19b6cca6a99d35ab3b90ad"
        },
        "date": 1756193221394,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20651,
            "unit": "ns/op\t    4888 B/op\t      71 allocs/op",
            "extra": "60464 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20651,
            "unit": "ns/op",
            "extra": "60464 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4888,
            "unit": "B/op",
            "extra": "60464 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60464 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20756,
            "unit": "ns/op\t    4899 B/op\t      71 allocs/op",
            "extra": "60694 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20756,
            "unit": "ns/op",
            "extra": "60694 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4899,
            "unit": "B/op",
            "extra": "60694 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60694 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "2045ea0c91f6f3ff8adae76601eb81cbb1588a38",
          "message": "🧹 Bump cnquery to v11.69.0 (#1804)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-08-26T08:24:05Z",
          "tree_id": "bc78eaa23642bc26eff6113cc1753b891281340c",
          "url": "https://github.com/mondoohq/cnspec/commit/2045ea0c91f6f3ff8adae76601eb81cbb1588a38"
        },
        "date": 1756196816346,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 22794,
            "unit": "ns/op\t    4899 B/op\t      71 allocs/op",
            "extra": "59439 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 22794,
            "unit": "ns/op",
            "extra": "59439 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4899,
            "unit": "B/op",
            "extra": "59439 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59439 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 22548,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "47587 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 22548,
            "unit": "ns/op",
            "extra": "47587 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "47587 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "47587 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "29d45991446d752b03691f38fb68e8b350aed1ec",
          "message": "🧹 Bump cnquery to v11.69.1 (#1806)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-08-26T22:11:53Z",
          "tree_id": "b1280422858104771215b30080f25e3d207f4e82",
          "url": "https://github.com/mondoohq/cnspec/commit/29d45991446d752b03691f38fb68e8b350aed1ec"
        },
        "date": 1756246480201,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19678,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "60565 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19678,
            "unit": "ns/op",
            "extra": "60565 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "60565 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60565 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20218,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "58969 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20218,
            "unit": "ns/op",
            "extra": "58969 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "58969 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58969 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "0a46a34ce3ebca5df08098c494e9f856e40f4748",
          "message": "🎉 v12 🎉",
          "timestamp": "2025-08-27T21:11:05Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1799/commits/0a46a34ce3ebca5df08098c494e9f856e40f4748"
        },
        "date": 1756443153903,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19851,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "56468 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19851,
            "unit": "ns/op",
            "extra": "56468 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "56468 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56468 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20111,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "63788 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20111,
            "unit": "ns/op",
            "extra": "63788 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "63788 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63788 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "289eeca4ae81fbf53e5cfa50bc4634b2b866c02d",
          "message": "🎉 v12 🎉",
          "timestamp": "2025-08-27T21:11:05Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1799/commits/289eeca4ae81fbf53e5cfa50bc4634b2b866c02d"
        },
        "date": 1756443925646,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20320,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "49816 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20320,
            "unit": "ns/op",
            "extra": "49816 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "49816 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "49816 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19980,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "58671 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19980,
            "unit": "ns/op",
            "extra": "58671 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "58671 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58671 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "10d82f404adbf71baa7f448df0d91296ebb11142",
          "message": "🎉 v12 🎉",
          "timestamp": "2025-08-27T21:11:05Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1799/commits/10d82f404adbf71baa7f448df0d91296ebb11142"
        },
        "date": 1756444460322,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19546,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "65806 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19546,
            "unit": "ns/op",
            "extra": "65806 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "65806 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "65806 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20582,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "58928 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20582,
            "unit": "ns/op",
            "extra": "58928 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "58928 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58928 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "b490bb077a288814692f1fddbedc0929905da142",
          "message": "🎉 v12 🎉",
          "timestamp": "2025-08-27T21:11:05Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1799/commits/b490bb077a288814692f1fddbedc0929905da142"
        },
        "date": 1756444645540,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 18946,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "68611 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 18946,
            "unit": "ns/op",
            "extra": "68611 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "68611 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "68611 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19887,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "64778 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19887,
            "unit": "ns/op",
            "extra": "64778 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "64778 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64778 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "dominik.richter@gmail.com",
            "name": "Dominik Richter",
            "username": "arlimus"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "6c2879b82eff36c6191a3539533a47f1c1a28ae2",
          "message": "🎉 v12.0.0-pre1 (#1799)\n\n\n\nSigned-off-by: Dominik Richter <dominik.richter@gmail.com>",
          "timestamp": "2025-08-28T22:26:18-07:00",
          "tree_id": "dbedb457640fc6f95a6031c68afbabcbe049ce2d",
          "url": "https://github.com/mondoohq/cnspec/commit/6c2879b82eff36c6191a3539533a47f1c1a28ae2"
        },
        "date": 1756445346549,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20889,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "58707 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20889,
            "unit": "ns/op",
            "extra": "58707 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "58707 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58707 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 22995,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "55088 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 22995,
            "unit": "ns/op",
            "extra": "55088 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "55088 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "55088 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "16705a88276f6387dc3979c0e61dd683185ce5ca",
          "message": "⭐ migrate --score-threshold to --risk-threshold",
          "timestamp": "2025-08-29T13:16:32Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1808/commits/16705a88276f6387dc3979c0e61dd683185ce5ca"
        },
        "date": 1756688551890,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21360,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "56523 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21360,
            "unit": "ns/op",
            "extra": "56523 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "56523 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56523 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20072,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "57045 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20072,
            "unit": "ns/op",
            "extra": "57045 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "57045 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57045 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "bce107500070e4141be6917d6b385899f7cd9f4c",
          "message": "⭐ migrate --score-threshold to --risk-threshold",
          "timestamp": "2025-08-29T13:16:32Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1808/commits/bce107500070e4141be6917d6b385899f7cd9f4c"
        },
        "date": 1756704566886,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20289,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "60409 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20289,
            "unit": "ns/op",
            "extra": "60409 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "60409 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60409 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19421,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "61440 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19421,
            "unit": "ns/op",
            "extra": "61440 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "61440 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61440 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "328b731bb95cb374fde31edc7d32e8c7868b7d07",
          "message": "⭐ migrate --score-threshold to --risk-threshold",
          "timestamp": "2025-08-29T13:16:32Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1808/commits/328b731bb95cb374fde31edc7d32e8c7868b7d07"
        },
        "date": 1756706041953,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20670,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "57000 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20670,
            "unit": "ns/op",
            "extra": "57000 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "57000 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57000 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 23038,
            "unit": "ns/op\t    4891 B/op\t      71 allocs/op",
            "extra": "57004 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 23038,
            "unit": "ns/op",
            "extra": "57004 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4891,
            "unit": "B/op",
            "extra": "57004 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57004 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "c241d3c3d26012171915396b16b1d2c7fb627cd4",
          "message": "⭐ migrate --score-threshold to --risk-threshold",
          "timestamp": "2025-08-29T13:16:32Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1808/commits/c241d3c3d26012171915396b16b1d2c7fb627cd4"
        },
        "date": 1756706926375,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21136,
            "unit": "ns/op\t    4889 B/op\t      71 allocs/op",
            "extra": "59025 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21136,
            "unit": "ns/op",
            "extra": "59025 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4889,
            "unit": "B/op",
            "extra": "59025 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59025 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20831,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "67075 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20831,
            "unit": "ns/op",
            "extra": "67075 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "67075 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "67075 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "328b731bb95cb374fde31edc7d32e8c7868b7d07",
          "message": "⭐ migrate --score-threshold to --risk-threshold",
          "timestamp": "2025-08-29T13:16:32Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1808/commits/328b731bb95cb374fde31edc7d32e8c7868b7d07"
        },
        "date": 1756708511771,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19924,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "54766 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19924,
            "unit": "ns/op",
            "extra": "54766 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "54766 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "54766 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20912,
            "unit": "ns/op\t    4890 B/op\t      71 allocs/op",
            "extra": "60002 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20912,
            "unit": "ns/op",
            "extra": "60002 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4890,
            "unit": "B/op",
            "extra": "60002 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60002 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "5f1e6076881d8976e4ec50983a31156676ff6dc7",
          "message": "⭐ migrate --score-threshold to --risk-threshold",
          "timestamp": "2025-08-29T13:16:32Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1808/commits/5f1e6076881d8976e4ec50983a31156676ff6dc7"
        },
        "date": 1756708898356,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20199,
            "unit": "ns/op\t    4899 B/op\t      71 allocs/op",
            "extra": "56152 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20199,
            "unit": "ns/op",
            "extra": "56152 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4899,
            "unit": "B/op",
            "extra": "56152 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56152 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19663,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "70028 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19663,
            "unit": "ns/op",
            "extra": "70028 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "70028 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "70028 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "b7264068e7262645d265b82bd0ce1bab9a20dd41",
          "message": "⭐ migrate --score-threshold to --risk-threshold",
          "timestamp": "2025-08-29T13:16:32Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1808/commits/b7264068e7262645d265b82bd0ce1bab9a20dd41"
        },
        "date": 1756711757352,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19550,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "61428 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19550,
            "unit": "ns/op",
            "extra": "61428 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "61428 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61428 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19643,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "53647 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19643,
            "unit": "ns/op",
            "extra": "53647 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "53647 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "53647 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "8a17b62fd489232827552578b97c23ee9ef607f7",
          "message": "🌈 streamline CLI output on risk score",
          "timestamp": "2025-08-29T13:16:32Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1809/commits/8a17b62fd489232827552578b97c23ee9ef607f7"
        },
        "date": 1756717882154,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19694,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "63163 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19694,
            "unit": "ns/op",
            "extra": "63163 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "63163 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63163 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19724,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "62102 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19724,
            "unit": "ns/op",
            "extra": "62102 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "62102 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62102 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "2669731715512a3948177b3a688c5a0156e6993e",
          "message": "Bump the gomodupdates group with 5 updates",
          "timestamp": "2025-08-29T13:16:32Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1812/commits/2669731715512a3948177b3a688c5a0156e6993e"
        },
        "date": 1756733261478,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21217,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "55420 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21217,
            "unit": "ns/op",
            "extra": "55420 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "55420 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "55420 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20032,
            "unit": "ns/op\t    4900 B/op\t      71 allocs/op",
            "extra": "58545 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20032,
            "unit": "ns/op",
            "extra": "58545 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4900,
            "unit": "B/op",
            "extra": "58545 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58545 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "ebf2e09af179527f470d54083dfe1966267f7ea0",
          "message": "🌈 streamline CLI output on risk score",
          "timestamp": "2025-08-29T13:16:32Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1809/commits/ebf2e09af179527f470d54083dfe1966267f7ea0"
        },
        "date": 1756743391518,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20659,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "56604 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20659,
            "unit": "ns/op",
            "extra": "56604 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "56604 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56604 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20052,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "61834 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20052,
            "unit": "ns/op",
            "extra": "61834 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "61834 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61834 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "dominik.richter@gmail.com",
            "name": "Dominik Richter",
            "username": "arlimus"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "d42fc89487ed27401f6e552b7f3f0d5146dd8d47",
          "message": "⭐ migrate --score-threshold to --risk-threshold (#1808)\n\nSigned-off-by: Dominik Richter <dominik.richter@gmail.com>",
          "timestamp": "2025-09-01T10:57:49-07:00",
          "tree_id": "8c2f5f92c959a757984dfa5a38c7724649b4fa65",
          "url": "https://github.com/mondoohq/cnspec/commit/d42fc89487ed27401f6e552b7f3f0d5146dd8d47"
        },
        "date": 1756749509486,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20813,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "62236 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20813,
            "unit": "ns/op",
            "extra": "62236 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "62236 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62236 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19654,
            "unit": "ns/op\t    4899 B/op\t      71 allocs/op",
            "extra": "64918 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19654,
            "unit": "ns/op",
            "extra": "64918 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4899,
            "unit": "B/op",
            "extra": "64918 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64918 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "409071e3ff827cf4bbfdb425d42936542bb2d87d",
          "message": "🌈 streamline CLI output on risk score",
          "timestamp": "2025-09-01T17:57:53Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1809/commits/409071e3ff827cf4bbfdb425d42936542bb2d87d"
        },
        "date": 1756749557791,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20488,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "57244 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20488,
            "unit": "ns/op",
            "extra": "57244 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "57244 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57244 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19222,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "59156 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19222,
            "unit": "ns/op",
            "extra": "59156 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "59156 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59156 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "dominik.richter@gmail.com",
            "name": "Dominik Richter",
            "username": "arlimus"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "46a459335da5a1a8dc98138a601d8300c5f42c7f",
          "message": "🌈 streamline CLI output on risk score (#1809)\n\n* 🌈 streamline CLI output on risk score\n\nSigned-off-by: Dominik Richter <dominik.richter@gmail.com>",
          "timestamp": "2025-09-01T11:20:53-07:00",
          "tree_id": "1817c4dff95c2394198e19d1bf644863827cb460",
          "url": "https://github.com/mondoohq/cnspec/commit/46a459335da5a1a8dc98138a601d8300c5f42c7f"
        },
        "date": 1756750889256,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 22575,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "55897 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 22575,
            "unit": "ns/op",
            "extra": "55897 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "55897 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "55897 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 22622,
            "unit": "ns/op\t    4891 B/op\t      71 allocs/op",
            "extra": "59146 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 22622,
            "unit": "ns/op",
            "extra": "59146 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4891,
            "unit": "B/op",
            "extra": "59146 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59146 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "4af7c06f3ceefa987dabe393db2aa7a4aa9ac131",
          "message": "⭐ add support for server-side features",
          "timestamp": "2025-09-01T18:20:57Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1814/commits/4af7c06f3ceefa987dabe393db2aa7a4aa9ac131"
        },
        "date": 1756751731279,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19938,
            "unit": "ns/op\t    4900 B/op\t      71 allocs/op",
            "extra": "68136 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19938,
            "unit": "ns/op",
            "extra": "68136 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4900,
            "unit": "B/op",
            "extra": "68136 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "68136 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19248,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "54798 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19248,
            "unit": "ns/op",
            "extra": "54798 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "54798 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "54798 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "dominik.richter@gmail.com",
            "name": "Dominik Richter",
            "username": "arlimus"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "999cf175454413d8f5e6645cc45d3c4c0ee1db3c",
          "message": "⭐ add support for server-side features (#1814)\n\nThis allows the server to set or unset certain features that it supports. This means e.g. that if the server can store resources data in the new format, we can send it this way.\n\nSigned-off-by: Dominik Richter <dominik.richter@gmail.com>",
          "timestamp": "2025-09-01T11:43:59-07:00",
          "tree_id": "d0701d1f32b0a0e0dade843f0ea9f3ac81ea86fe",
          "url": "https://github.com/mondoohq/cnspec/commit/999cf175454413d8f5e6645cc45d3c4c0ee1db3c"
        },
        "date": 1756752278067,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19818,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "63526 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19818,
            "unit": "ns/op",
            "extra": "63526 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "63526 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63526 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19866,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "63138 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19866,
            "unit": "ns/op",
            "extra": "63138 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "63138 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63138 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "ea3a61805a257efed7c5525963a0b600664a9b79",
          "message": "🦘 update cnquery v12-pre3",
          "timestamp": "2025-09-01T18:44:03Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1815/commits/ea3a61805a257efed7c5525963a0b600664a9b79"
        },
        "date": 1756775384981,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20360,
            "unit": "ns/op\t    4890 B/op\t      71 allocs/op",
            "extra": "56732 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20360,
            "unit": "ns/op",
            "extra": "56732 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4890,
            "unit": "B/op",
            "extra": "56732 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56732 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19449,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "62479 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19449,
            "unit": "ns/op",
            "extra": "62479 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "62479 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62479 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "dominik.richter@gmail.com",
            "name": "Dominik Richter",
            "username": "arlimus"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "a9edb21e4d7bae238f78bc2862737b7911dd7380",
          "message": "🦘 update cnquery v12-pre3 (#1815)\n\nSigned-off-by: Dominik Richter <dominik.richter@gmail.com>",
          "timestamp": "2025-09-01T21:27:58-07:00",
          "tree_id": "a3a7077bffcadf20b036349c957e7bcbd73e1380",
          "url": "https://github.com/mondoohq/cnspec/commit/a9edb21e4d7bae238f78bc2862737b7911dd7380"
        },
        "date": 1756787446425,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20620,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "57031 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20620,
            "unit": "ns/op",
            "extra": "57031 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "57031 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57031 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19851,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "61905 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19851,
            "unit": "ns/op",
            "extra": "61905 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "61905 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61905 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "6195e82170490aff3466f7f3bd5a5a2ec7289cce",
          "message": "🦘 v12.0.0-rc1",
          "timestamp": "2025-09-02T04:28:01Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1817/commits/6195e82170490aff3466f7f3bd5a5a2ec7289cce"
        },
        "date": 1756801104265,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20454,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "62412 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20454,
            "unit": "ns/op",
            "extra": "62412 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "62412 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62412 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 22028,
            "unit": "ns/op\t    4890 B/op\t      71 allocs/op",
            "extra": "53211 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 22028,
            "unit": "ns/op",
            "extra": "53211 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4890,
            "unit": "B/op",
            "extra": "53211 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "53211 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "dominik.richter@gmail.com",
            "name": "Dominik Richter",
            "username": "arlimus"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "b633bb7f22c969621ceeca7b4c1abb60251a577f",
          "message": "🦘 v12.0.0-rc1 (#1817)\n\nSigned-off-by: Dominik Richter <dominik.richter@gmail.com>",
          "timestamp": "2025-09-02T01:24:33-07:00",
          "tree_id": "dd437159143a05e701d78e923896af47325a61d7",
          "url": "https://github.com/mondoohq/cnspec/commit/b633bb7f22c969621ceeca7b4c1abb60251a577f"
        },
        "date": 1756801640690,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20214,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "55128 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20214,
            "unit": "ns/op",
            "extra": "55128 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "55128 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "55128 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20462,
            "unit": "ns/op\t    4902 B/op\t      71 allocs/op",
            "extra": "58803 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20462,
            "unit": "ns/op",
            "extra": "58803 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4902,
            "unit": "B/op",
            "extra": "58803 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58803 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "7cc010d079b9d803ea28b6941c042926b1de8fb0",
          "message": "⭐ policy require providers",
          "timestamp": "2025-09-02T08:55:45Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1803/commits/7cc010d079b9d803ea28b6941c042926b1de8fb0"
        },
        "date": 1756855625560,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19326,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "67244 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19326,
            "unit": "ns/op",
            "extra": "67244 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "67244 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "67244 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20059,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "58533 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20059,
            "unit": "ns/op",
            "extra": "58533 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "58533 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58533 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "0a4fb98d16d4f6bd774220acc3cf3bda6585ca69",
          "message": "Bump the gomodupdates group across 1 directory with 6 updates",
          "timestamp": "2025-09-02T08:55:45Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1819/commits/0a4fb98d16d4f6bd774220acc3cf3bda6585ca69"
        },
        "date": 1756857797547,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19912,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "57866 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19912,
            "unit": "ns/op",
            "extra": "57866 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "57866 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57866 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20987,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "57976 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20987,
            "unit": "ns/op",
            "extra": "57976 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "57976 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57976 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "5cacb5af73e96917aee432342f88950e1c3ae2e0",
          "message": "🧹 Bump cnquery to v12.0.0 (#1826)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-09-05T16:36:18Z",
          "tree_id": "b0896a050767514ec82da71248607468f7236119",
          "url": "https://github.com/mondoohq/cnspec/commit/5cacb5af73e96917aee432342f88950e1c3ae2e0"
        },
        "date": 1757090349576,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 18669,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "57822 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 18669,
            "unit": "ns/op",
            "extra": "57822 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "57822 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57822 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20851,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "55034 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20851,
            "unit": "ns/op",
            "extra": "55034 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "55034 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "55034 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "6c3a695a193f3635020fb0a0cf3fa345fe993f59",
          "message": "Bump actions/setup-go from 5 to 6",
          "timestamp": "2025-09-05T16:36:22Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1828/commits/6c3a695a193f3635020fb0a0cf3fa345fe993f59"
        },
        "date": 1757315543486,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19144,
            "unit": "ns/op\t    4891 B/op\t      71 allocs/op",
            "extra": "60800 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19144,
            "unit": "ns/op",
            "extra": "60800 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4891,
            "unit": "B/op",
            "extra": "60800 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60800 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19741,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "61234 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19741,
            "unit": "ns/op",
            "extra": "61234 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "61234 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61234 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "7bcd902e5a5a86ec3ab2b2a632c6e02c709b1aff",
          "message": "Bump the gomodupdates group with 3 updates",
          "timestamp": "2025-09-05T16:36:22Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1829/commits/7bcd902e5a5a86ec3ab2b2a632c6e02c709b1aff"
        },
        "date": 1757315762525,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 18787,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "61917 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 18787,
            "unit": "ns/op",
            "extra": "61917 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "61917 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "61917 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 22634,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "62116 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 22634,
            "unit": "ns/op",
            "extra": "62116 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "62116 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62116 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "4780fe7a4534ddbaa2ed13ed3ea0147ed15a5e76",
          "message": "Bump the gomodupdates group across 1 directory with 8 updates",
          "timestamp": "2025-09-05T16:36:22Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1830/commits/4780fe7a4534ddbaa2ed13ed3ea0147ed15a5e76"
        },
        "date": 1757921460642,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19859,
            "unit": "ns/op\t    4896 B/op\t      71 allocs/op",
            "extra": "56042 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19859,
            "unit": "ns/op",
            "extra": "56042 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4896,
            "unit": "B/op",
            "extra": "56042 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56042 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20316,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "54787 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20316,
            "unit": "ns/op",
            "extra": "54787 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "54787 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "54787 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "ad32d1d69a2514e618b73520d658c2ed10648e12",
          "message": "🧹 Bump cnquery to v12.1.0 (#1831)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-09-16T09:38:22Z",
          "tree_id": "f6d93a17c4b0d4355921bcc758506b02b99444ce",
          "url": "https://github.com/mondoohq/cnspec/commit/ad32d1d69a2514e618b73520d658c2ed10648e12"
        },
        "date": 1758015668481,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 18767,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "59205 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 18767,
            "unit": "ns/op",
            "extra": "59205 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "59205 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59205 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 18644,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "60050 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 18644,
            "unit": "ns/op",
            "extra": "60050 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "60050 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60050 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "7662d4cd974736580cd581076cc6459227473146",
          "message": "✨ extend exception review status",
          "timestamp": "2025-09-16T09:38:27Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1833/commits/7662d4cd974736580cd581076cc6459227473146"
        },
        "date": 1758033301649,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 21100,
            "unit": "ns/op\t    4889 B/op\t      71 allocs/op",
            "extra": "58666 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 21100,
            "unit": "ns/op",
            "extra": "58666 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4889,
            "unit": "B/op",
            "extra": "58666 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58666 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 22168,
            "unit": "ns/op\t    4891 B/op\t      71 allocs/op",
            "extra": "50647 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 22168,
            "unit": "ns/op",
            "extra": "50647 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4891,
            "unit": "B/op",
            "extra": "50647 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "50647 times\n4 procs"
          }
        ]
      }
    ]
  }
}