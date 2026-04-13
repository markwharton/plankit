# Plan: Restructure resources.md with Well-Architected Framework section

## Context

The current `docs/resources.md` has AWS and Azure sections at the top with deployment-specific links. The user wants to replace them with a balanced "Well-Architected Framework" section at the end covering AWS, Azure, and Google Cloud. Deployment-specific links move to `creating-skills.md` where they have context.

## Files to modify

- `docs/resources.md` — remove AWS/Azure sections, add Well-Architected Framework at end
- `docs/creating-skills.md` — clean up out-of-place references, link preview platforms

## resources.md changes

**Remove:** AWS and Azure sections (lines 3-13) with all deployment-specific links.

**New section order:** Claude Code, Git, GitHub CLI, Well-Architected Framework

**Add at end:** Well-Architected Framework section with:
1. Brief intro explaining the common pillars across all three providers
2. Pillar comparison table (visual guide)
3. Per-provider subsections — each gets a framework link + pillar links (balanced)

### Final resources.md Well-Architected section:

```markdown
## Well-Architected Framework

AWS, Azure, and Google Cloud each publish a Well-Architected Framework — a structured set of best practices organized around the same core pillars. The names differ slightly but the concepts align. The patterns transfer across providers.

| Pillar | AWS | Azure | Google Cloud |
|--------|-----|-------|-------------|
| Operations | Operational Excellence | Operational Excellence | Operational Excellence |
| Security | Security | Security | Security, Privacy, and Compliance |
| Reliability | Reliability | Reliability | Reliability |
| Performance | Performance Efficiency | Performance Efficiency | Performance Optimization |
| Cost | Cost Optimization | Cost Optimization | Cost Optimization |
| Sustainability | Sustainability | — | Sustainability |

### AWS

- [AWS Well-Architected Framework](https://docs.aws.amazon.com/wellarchitected/latest/framework/welcome.html)
- Pillars: [Operational Excellence](https://docs.aws.amazon.com/wellarchitected/latest/operational-excellence-pillar/welcome.html) | [Security](https://docs.aws.amazon.com/wellarchitected/latest/security-pillar/welcome.html) | [Reliability](https://docs.aws.amazon.com/wellarchitected/latest/reliability-pillar/welcome.html) | [Performance Efficiency](https://docs.aws.amazon.com/wellarchitected/latest/performance-efficiency-pillar/welcome.html) | [Cost Optimization](https://docs.aws.amazon.com/wellarchitected/latest/cost-optimization-pillar/welcome.html) | [Sustainability](https://docs.aws.amazon.com/wellarchitected/latest/sustainability-pillar/sustainability-pillar.html)

### Azure

- [Azure Well-Architected Framework](https://learn.microsoft.com/en-us/azure/well-architected/)
- Pillars: [Reliability](https://learn.microsoft.com/en-us/azure/well-architected/reliability/) | [Security](https://learn.microsoft.com/en-us/azure/well-architected/security/) | [Cost Optimization](https://learn.microsoft.com/en-us/azure/well-architected/cost-optimization/) | [Operational Excellence](https://learn.microsoft.com/en-us/azure/well-architected/operational-excellence/) | [Performance Efficiency](https://learn.microsoft.com/en-us/azure/well-architected/performance-efficiency/)

### Google Cloud

- [Google Cloud Well-Architected Framework](https://docs.cloud.google.com/architecture/framework)
- Pillars: [Operational Excellence](https://docs.cloud.google.com/architecture/framework/operational-excellence) | [Security, Privacy, and Compliance](https://docs.cloud.google.com/architecture/framework/security) | [Reliability](https://docs.cloud.google.com/architecture/framework/reliability) | [Performance Optimization](https://docs.cloud.google.com/architecture/framework/performance-optimization) | [Cost Optimization](https://docs.cloud.google.com/architecture/framework/cost-optimization) | [Sustainability](https://docs.cloud.google.com/architecture/framework/sustainability)
```

## creating-skills.md changes

1. **Link preview platforms** in /preview tutorial intro (line 125) — make the named platforms clickable:
   - [Azure Static Web Apps](https://learn.microsoft.com/en-us/azure/static-web-apps/preview-environments)
   - [Netlify](https://docs.netlify.com/site-deploys/deploy-previews/)
   - [Vercel](https://vercel.com/docs/deployments/preview-deployments)
   - GitHub Pages stays plain text (no native PR preview)

2. **Remove** the `[AWS Well-Architected Framework]` line from the References section

3. **Remove** the "Schema and data changes" variation from /rollback Variations to consider

## Verification

- resources.md section order: Claude Code, Git, GitHub CLI, Well-Architected Framework
- All three providers balanced — framework link + pillar links each
- creating-skills.md: preview platforms linked, no architecture refs in References, no schema variation
- Deployment-specific links (rollback safety, backwards compatibility) remain in creating-skills.md /rollback variations
- Amend into the existing unpushed commit
