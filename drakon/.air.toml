root = "/app"

[build]
cmd = 'go build -gcflags "all=-N -l" -o delvebuild .'
bin = "/app/delvebuild"
full_bin = "dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec /app/delvebuild"
watch = ["./..."]
include_ext = ["go", "tpl", "tmpl", "html"]
