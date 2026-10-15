# 2026-10-15 — Box snapshot restore and provenance gap (~10:29 CT)

## Incident

A **box snapshot restore** on **2026-10-15 at approximately 10:29 CT** destroyed the in-progress **v6 freeze** artifacts: split definition, local **`.hbc`** files, and on-disk **triples**. Ground-truth labels **survived** the restore.

## Provenance gap: `bin/hedis-m37`

The binary originally referenced as **`36d63cb0…`** could **not** be reproduced from recorded inputs. We therefore treat that artifact as a **provenance gap** and do not cite metrics tied exclusively to it.

We rebuilt **`bin/hedis-m37`** at **`647bfead…`** from frozen matcher **`359ec3f`**, which again yielded real-dev **0.840** and synthetic **0.970** on the development split.

## v6 re-freeze

We re-established v6 from the same application tags:

| Artifact | Reference |
|----------|-----------|
| Branch `research/v6-freeze` | `4eb9220a` |
| Manifest | `91a5069` |

## Ground-truth drift

**Pixelfed** ground truth grew from **62** to **236** packages after a **StoryCarousel** stub entered the bundled graph. **lxmusic** triples remained **byte-identical** under both dump pipelines we compared.

## Decision

We will not treat the pre-restore v6 tree as recoverable; scoring uses the re-frozen manifest only.

## Next steps

- [ ] Score v6 once under M37 (see [2026-10-15-v6-held-out.md](./2026-10-15-v6-held-out.md)).
