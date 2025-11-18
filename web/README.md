# Web Package

Embeds static web assets (HTML, CSS, JavaScript) into the Go binary using native `//go:embed`.

## Structure

```
web/
├── embed.go          # Embeds static/ directory using //go:embed
└── static/           # Source files: index.html, style.css, app.js
```

## Usage

The `Static` variable is an `embed.FS` that contains all files from `static/`:

```go
import "github.com/meedamian/fat/web"

// web.Static contains embedded static files
// Access them like: web.Static.ReadFile("static/index.html")
```

## How It Works

Go's native `//go:embed` directive (Go 1.16+) includes files in the compiled binary at build time:

```go
//go:embed static
var Static embed.FS
```

This is exactly how `internal/constants/questions.go` embeds `questions.txt`.

## Benefits

✅ **Native Go** - No build scripts, Makefiles, or external tools  
✅ **Single binary** - All assets included, ~55MB total  
✅ **Type-safe** - `embed.FS` is a standard library interface  
✅ **Simple** - Just `go build` and you're done  

## Deployment

The compiled binary is standalone. Copy it anywhere and run:

```bash
scp fat user@server:/opt/fat
ssh user@server
cd /opt/fat
./fat  # Just works!
```

Only optional dependency: `.env` file for API keys (can also use environment variables).
