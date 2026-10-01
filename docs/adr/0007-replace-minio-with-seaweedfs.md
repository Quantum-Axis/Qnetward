# 7. Replace MinIO with SeaweedFS in the production stack

## Status

Accepted

## Context

`deploy/docker-compose.production.yml` bundled `minio/minio:latest` as the
S3-compatible object storage backing `NETWATCHER_S3_*`. The `docker-smoke`
CI job, which builds and runs this exact compose file, started failing
with `pull access denied for minio/minio, repository does not exist`.

This was not a transient registry outage. MinIO archived the community
edition of their server in April 2026 and moved to source-only
distribution; Docker Hub subsequently removed the `minio` namespace
entirely, with pulls failing between September 10–13, 2026. The
community's standard workaround — repointing at `quay.io/minio/minio`,
which had mirrored the same images — worked for thirteen days before quay
gated anonymous pulls too; as of September 24, 2026, neither registry
serves `minio/minio` without authentication MinIO does not offer for the
archived community edition. This is a dead end, not a registry hiccup to
wait out: there is no container image to pin to, official or mirrored.

Replacing a load-bearing dependency's container image with an unofficial
or community-rebuilt mirror would trade one disappearing-image risk for
another (no governance, no guarantee it keeps being updated or stays
available), so that was ruled out rather than evaluated further.

## Decision

Replace MinIO with [SeaweedFS](https://github.com/seaweedfs/seaweedfs)
(`chrislusf/seaweedfs`), an Apache-2.0, actively maintained, S3-compatible
object store that still publishes official container images. Internally,
NetWatcher only ever talks to its object storage through
`github.com/minio/minio-go/v7` as a generic S3 API client (bucket
exists/create, put object, presigned GET) — that client library is
unrelated to which server backs it and needed no changes.

The image's default command, `weed mini -dir=/data`, runs an all-in-one
master+volume+filer+S3-gateway process — functionally the same role
MinIO's single `server /data` command played — and reads
`AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` as static S3 credentials with
no extra setup step, the same zero-config experience
`MINIO_ROOT_USER`/`MINIO_ROOT_PASSWORD` gave. The S3 API listens on
`:8333` (SeaweedFS's default) rather than MinIO's `:9000`.

Renamed to make the swap legible rather than leaving MinIO-flavored names
pointing at a different product:

- Compose service `minio` → `objectstore`; volume `minio-data` →
  `objectstore-data`.
- Env vars `MINIO_ROOT_USER`/`MINIO_ROOT_PASSWORD` →
  `OBJECTSTORE_ACCESS_KEY`/`OBJECTSTORE_SECRET_KEY`, updated in
  `.env.example`, `.github/workflows/ci.yml`, and the README.
- `NETWATCHER_S3_ENDPOINT` changed from `minio:9000` to
  `objectstore:8333` in both the compose file and README.

`NETWATCHER_S3_*` itself is unchanged — it was already a generic
S3-compatible configuration surface, never MinIO-specific.

## Consequences

- The `docker-smoke` CI job and the production compose stack both build
  against an object storage backend with a real, currently-maintained
  container image, closing the gap this ADR was written to fix.
- Anyone with an existing `.env` using `MINIO_ROOT_USER`/
  `MINIO_ROOT_PASSWORD` needs to rename those keys to
  `OBJECTSTORE_ACCESS_KEY`/`OBJECTSTORE_SECRET_KEY` (same values, new
  names) when picking up this change.
- SeaweedFS's S3 gateway is not a byte-for-byte MinIO clone; it implements
  the S3 API surface NetWatcher actually uses (bucket exists/create, put
  object, presigned GET), but a future feature that depends on
  MinIO-specific behavior (e.g. its admin API or specific extensions)
  would need re-evaluating against SeaweedFS's actual API coverage.
- Object storage remains optional in the application itself
  (`objectstore.Open` returns `nil, nil` when `NETWATCHER_S3_ENDPOINT` is
  empty) — this ADR only changes what the bundled production stack
  provides when someone wants it enabled out of the box.
