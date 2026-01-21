# Contributing to cnspec Documentation

This folder contains the user-facing documentation for cnspec. The content is synced to the [Mondoo docs site](https://mondoo.com/docs/cnspec/).

## How the sync works

The cnspec docs live in this open source repo but are published on mondoo.com/docs. A GitHub Action in the docs repo syncs the content:

1. **Source**: This folder (`cnspec/docs/`)
2. **Destination**: `docs/content/cnspec/` in the private docs repo
3. **Trigger**: Daily at 6 AM UTC or manual dispatch

### What gets synced

- All `.mdx` files (documentation content)
- CLI reference files in `cli/` folder (auto-generated)

### What stays in the docs repo only

- `meta.json` files (fumadocs navigation configuration)
- Images in `public/img/cnspec/`

The sync workflow backs up and restores `meta.json` files to preserve navigation structure.

## Writing documentation

### File format

- Use `.mdx` extension for all documentation files
- CLI reference files in `cli/` use `.md` (auto-generated)

### Frontmatter

Each file needs frontmatter with at least:

```yaml
---
title: Page Title
sidebar_label: Short Label
displayed_sidebar: cnspec
description: Brief description for SEO
image: /img/cnspec/mondoo-feature.jpg
---
```

### Images

Images are stored in the docs repo at `public/img/cnspec/`. Reference them with absolute paths:

```markdown
![Alt text](/img/cnspec/folder/image.png)
```

Available image folders:

- `/img/cnspec/` - Featured images, banners
- `/img/cnspec/aws/` - AWS screenshots
- `/img/cnspec/github/` - GitHub app setup screenshots
- `/img/cnspec/gw/` - Google Workspace screenshots
- `/img/cnspec/m365/` - Microsoft 365 screenshots
- `/img/cnspec/terraform/` - Terraform screenshots

### Links

Use absolute paths for links:

```markdown
[MQL Reference](/mql/resources/)
[Other cnspec page](/cnspec/cnspec-about/)
```

## Adding new images

If you need new images:

1. Add the image to the docs repo at `public/img/cnspec/`
2. Reference it in your markdown with the absolute path
3. The image will be available after the docs repo change is merged

## Folder structure

```
docs/
├── cli/                 # Auto-generated CLI reference (do not edit)
├── cloud/               # Cloud provider docs (AWS, Azure, GCP, etc.)
│   ├── aws/
│   ├── gcp/
│   ├── k8s/
│   └── ...
├── cnspec-adv-install/  # Advanced installation guides
├── network/             # Network device docs
├── os/                  # Operating system docs
├── saas/                # SaaS integration docs (GitHub, M365, etc.)
├── supplychain/         # Supply chain security docs
├── write-policies/      # Policy authoring guides
├── index.mdx            # cnspec docs home page
├── cnspec-about.mdx     # What is cnspec
└── CONTRIBUTING.md      # This file
```
