window.BENCHMARK_DATA = {
  "lastUpdate": 1753717212547,
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
      }
    ]
  }
}