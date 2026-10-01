# Optional media module

Install with `make-app feature add media`. The isolated `modules/media` Go module
pins MinIO's S3 SDK and its complete dependency graph. Default applications acquire
no media dependency. Run `cd modules/media && GOWORK=off go test ./...` to verify it.
When integrating, add it to your workspace with `go work use modules/media` and
add the local module requirement/replacement to the API module for standalone
container builds. Review and commit those composition changes with your product
spec and endpoint tests.

The `blob` package owns storage and upload-completion ports. Bootstrap chooses
`adapters/filesystem` or `adapters/s3`. Filesystem storage uses Go's rooted file API,
refuses traversal and replacement, and bounds reads/writes. S3 uses TLS, configured
credentials, a bucket, and a strict byte limit. Use distinct immutable object keys
and bucket lifecycle policies; object names must never carry credentials.

Persist a trusted `blob.Upload` intent containing authorized owner, random object
key, distinct unpublished staging and final keys, exact size/type and expiry. The S3 adapter creates an expiring POST policy.
Inject `adapters/content.Inspector` at bootstrap for MIME detection.
After upload, `Completion.Complete` verifies the authenticated owner, expiry,
actual byte count and detected MIME type, then exclusively publishes to the separate final key and returns a checksum. Replaying the upload can change only the staging object, never the final object. Repeated completion refuses to overwrite the final key. Queue abandoned staging/final objects for cleanup if your attachment transaction fails. Do not trust
an upload intent supplied by the client. Commit the resulting attachment and audit
record in the application's transaction before exposing it. Owner checks here
supplement, rather than replace, the application's authorization port.

`imagework.Thumbnail` validates dimensions before decoding JPEG/PNG and emits a
metadata-free bounded PNG. It uses a simple nearest-neighbor resize; choose a
reviewed higher-quality image processor behind your port if the product needs it.
Persist deletion work in an outbox. `blob.DeleteBatch` handles a bounded claimed
batch and returns completed keys so failures can be retried. Recheck application
references before deleting a blob; this module cannot know product ownership.
