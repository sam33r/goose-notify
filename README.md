# goose-notify

macOS CLI that displays a black, fading notification toast near the top-center of the active screen, then exits.

## Usage

```
goose-notify "your message"
echo "your message" | goose-notify
goose-notify --title "Status" --duration 3s "Build succeeded"
```

## Build

```bash
make build         # build for host arch
make install       # universal binary to /usr/local/bin
make test
```

See [docs/superpowers/specs/2026-05-04-goose-notify-design.md](docs/superpowers/specs/2026-05-04-goose-notify-design.md) for the full design.
