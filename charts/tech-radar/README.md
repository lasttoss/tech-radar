# tech-radar chart

A CronJob, because that is what this program is: a batch that runs once a day,
produces one digest, and exits.

| value | default | why |
|---|---|---|
| `schedule` | `0 4 * * *` UTC | 11:00 in Vietnam, when the digest is actually read |
| `concurrencyPolicy` | `Forbid` | two overlapping runs would score the same day twice |
| `delivery.enabled` | `false` | dry run until a token is deliberately provided: the digest goes into the log |
| `telegram.existingSecret` | empty | bring your own secret; otherwise one is created from the values |
| `networkPolicy.enabled` | `false` | off because the network shape is cluster specific, but the template is here |

**No HPA and no PDB, on purpose.** There is nothing to scale in a job that runs for
a second, and a PodDisruptionBudget on a pod that exists to finish and exit would
block node drains for no benefit. The template list for this repository's other
charts contains both; copying them here would be cargo cult.

**Nothing inbound.** The pod has no port and the NetworkPolicy (when enabled) has no
ingress rule: this program only ever talks outwards.

```bash
helm lint --strict charts/tech-radar
helm template tech-radar charts/tech-radar | less

# with delivery on, tokens from a secret you manage elsewhere
helm upgrade --install tech-radar charts/tech-radar \
  --set delivery.enabled=true \
  --set telegram.existingSecret=radar-telegram
```
