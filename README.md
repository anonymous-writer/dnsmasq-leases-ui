# dnsmasq-leases-ui

[![GHCR](https://img.shields.io/badge/GHCR-docker%20image-blue?logo=github)](https://github.com/anonymous-writer/dnsmasq-leases-ui/pkgs/container/dnsmasq-leases-ui)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/anonymous-writer/dnsmasq-leases-ui)](LICENSE)

A lightweight web interface for the [dnsmasq](https://thekelleys.org.uk/dnsmasq/doc.html) DHCP lease file, written in Go. It provides a searchable, sortable view of DHCP leases, detects reachable HTTP/HTTPS interfaces on client devices, and lets you manage which devices have already been acknowledged.

![Screenshot](https://raw.githubusercontent.com/anonymous-writer/dnsmasq-leases-ui/main/docs/screenshot.png)

## Features

- Searchable and sortable DHCP lease table
- IPv4 and IPv6 support, with filters for IP version and DHCP reservations
- Manual refresh of the lease list; lease countdowns update locally in the browser
- Device details dialog with hostname, IP, MAC, reservation status, lease end, and detected web URLs
- Status indicator: green when HTTP/HTTPS responds, yellow when ping responds but no web interface is detected, red when neither check responds
- Automatically detects reachable HTTP/HTTPS interfaces and makes device IP addresses clickable
- Shows a clickable hostname when it resolves to the corresponding lease IP
- Detects DHCP reservations from the optional `dnsmasq.dhcphosts` file
- Displays infinite leases as `Never` and expired leases as `Expired`
- Detects newly seen devices by MAC address and marks them with a `NEW` indicator
- Persistently remembers acknowledged devices in `/data/known-macs.json`
- Per-device actions to mark a device as known or mark it as new again
- Bulk actions to mark all detected DHCP reservations as known or reset the remembered-device list
- IPv6 leases without a valid MAC address remain visible but are not marked as new
- Dark/light theme and responsive web interface
- JSON API at `/leases` and POST actions at `/known-macs`
- Bounded concurrent checks for device web interfaces
- Multi-architecture Docker image: `linux/amd64` and `linux/arm64`
- Runs as a non-root user in the provided Docker image

## Quick start

The current development image is published to GitHub Container Registry with the `go-beta` tag. Pull it with:

```bash
docker pull ghcr.io/anonymous-writer/dnsmasq-leases-ui:go-beta
```

The `latest` tag is intended for the stable release. The manual development workflow described under [Docker image publishing](#docker-image-publishing) updates `go-beta` only; it does not update `latest` when you publish a GitHub Release or Pre-release.

The container reads these files by default:

```text
/var/lib/dnsmasq/dnsmasq.leases
/var/lib/dnsmasq/dnsmasq.dhcphosts
```

The hosts file is optional. Mount the containing directory rather than individual files so the container continues to see the current lease file if `dnsmasq` replaces it.

### Docker

```bash
docker run -d \
  --name dnsmasq-leases-ui \
  --restart unless-stopped \
  --network host \
  -e PORT=3008 \
  -v /var/lib/dnsmasq:/var/lib/dnsmasq:ro \
  -v "$(pwd)/data:/data" \
  ghcr.io/anonymous-writer/dnsmasq-leases-ui:go-beta
```

Create the persistent data directory before starting the container:

```bash
mkdir -p data
```

Open `http://<host>:3008`.

For the stable image, replace `:go-beta` with `:latest`.

Host networking lets the app reach devices on the LAN, including IPv6 devices, when the host itself has the required routes. With host networking, do not use `-p`/`--publish`; the application listens directly on the host network.

### Docker Compose / Podman Compose

Example `compose.yaml`:

```yaml
services:
  dnsmasq-leases-ui:
    image: ghcr.io/anonymous-writer/dnsmasq-leases-ui:go-beta
    container_name: dnsmasq-leases-ui
    restart: unless-stopped
    environment:
      PORT: "3008"
      HOME: /tmp
      XDG_RUNTIME_DIR: /tmp
    network_mode: host
    volumes:
      - /var/lib/dnsmasq:/var/lib/dnsmasq:ro
      - ./data:/data:Z
```

Create the data directory before the first start:

```bash
mkdir -p data
```

Start the service with `docker compose up -d` or `podman compose up -d`.

The `:Z` suffix is useful with Podman on SELinux-enabled systems. For Docker, `./data:/data` can be used if SELinux relabeling is not required.

#### Rootless Podman: permissions for remembered devices

The application runs as a non-root user inside the container. The `/data` directory must be writable so the app can save `known-macs.json`. With the supplied image, the application user is normally UID `100` and GID `101`.

For rootless Podman, set ownership from inside Podman's user namespace:

```bash
cd /path/to/project
mkdir -p data
podman unshare chown -R 100:101 ./data
```

Then start or restart the service. This is a one-time ownership fix for the data directory; do not remove the directory to fix permissions, because it contains the remembered-device list.

You can test write access with:

```bash
podman exec dnsmasq-leases-ui sh -c \
  'id; ls -ldn /data; touch /data/testfile && rm /data/testfile'
```

If you use a different image or change its configured user, verify the UID/GID with `podman exec dnsmasq-leases-ui id` and adapt the ownership command accordingly.

### Configure dnsmasq

Configure dnsmasq to write its lease file to the mounted directory:

```ini
dhcp-leasefile=/var/lib/dnsmasq/dnsmasq.leases
```

If you use a separate DHCP hosts file, configure that too:

```ini
dhcp-hostsfile=/var/lib/dnsmasq/dnsmasq.dhcphosts
```

Create the directory before restarting dnsmasq:

```bash
sudo mkdir -p /var/lib/dnsmasq
```

After changing the configuration, restart dnsmasq and verify that the files exist and contain the expected data.

## Using the interface

- Use the search box to filter by hostname, IP, MAC address, lease time, status, or reservation state.
- Use the IP-version selector to show all leases, IPv4 only, or IPv6 only.
- Use the reservation selector to show all leases, reservations only, or dynamic leases only.
- Select a table heading to change the sort order.
- Click a row to open device details. Copy buttons are available for hostname, IP, and MAC values.
- Click **Refresh** to reload leases and redo device reachability checks. The countdown is updated locally between refreshes; reachability checks are not run every second.
- A `NEW` label means the MAC address is not currently in the remembered-device list.
- In device details, choose **Mark as known** to remember a device or **Mark as new** to remove its MAC from the remembered list.
- Choose **Mark reservations known** to remember all currently detected DHCP reservations that have valid MAC addresses.
- Choose **Reset remembered** to clear all remembered MAC addresses. The UI asks for confirmation first; devices with valid MAC addresses can be marked as new again.

The remembered list is stored in `known-macs.json` and persists across container restarts when `/data` is mounted to a persistent host directory. IPv6 leases without a valid MAC address are not marked as new because they cannot be reliably identified by the MAC-based list.

## Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | Web interface |
| `GET` | `/leases` | JSON response containing the leases array |
| `POST` | `/known-macs` | Mark/unmark MAC addresses or perform bulk actions |

Example:

```bash
curl http://localhost:3008/leases
```

Each object in the `leases` array can include `staticIP`, `leasetime`, `leaseExpiry`, `macAddress`, `ipAddress`, `name`, `webUrl`, `webHostUrl`, `status`, and `isNew`. `leaseExpiry` is the lease-expiry Unix timestamp in seconds; `0` indicates an infinite lease.

`POST /known-macs` accepts JSON. The `action` values are:

| Action | Payload example | Result |
|---|---|---|
| `mark` (default) | `{"mac":"AA:BB:CC:DD:EE:FF","action":"mark"}` | Remember one valid MAC address |
| `unmark` | `{"mac":"AA:BB:CC:DD:EE:FF","action":"unmark"}` | Remove one MAC address from the remembered list |
| `reset` | `{"action":"reset"}` | Clear the remembered list |
| `mark-reservations` | `{"action":"mark-reservations"}` | Remember all currently detected DHCP reservations with valid MAC addresses |

The `mark-reservations` response also includes a `count` field. These endpoints do not provide authentication; keep the application on a trusted network and do not expose it directly to the public internet.

## Configuration

| Environment variable | Default | Description |
|---|---|---|
| `DNSMASQ_LEASES_FILE` | `/var/lib/dnsmasq/dnsmasq.leases` | Path to the dnsmasq lease file |
| `DNSMASQ_HOSTS_FILE` | `/var/lib/dnsmasq/dnsmasq.dhcphosts` | Path to the optional DHCP hosts file |
| `KNOWN_MACS_FILE` | `/data/known-macs.json` | Path to the persistent remembered-MAC list |
| `HOST` | `0.0.0.0` | Address the web server listens on |
| `PORT` | `5000` | Port the web server listens on |
| `WEB_UI_TIMEOUT` | `500ms` | Timeout for an HTTP/HTTPS or hostname-resolution check; accepts Go durations such as `500ms` or seconds such as `0.5` |
| `PING_TIMEOUT` | `1200ms` | Timeout for a ping probe; accepts Go durations or a number of seconds |
| `WEB_UI_MAX_WORKERS` | `16` | Maximum concurrent IP web-interface checks |
| `APP_VERSION` | `dev` | Version shown by the application |
| `APP_RELEASE_DATE` | empty | Build/release date shown by the application |
| `REPO_URL` | `https://github.com/anonymous-writer/dnsmasq-leases-ui` | Repository URL shown in the UI |

The `dnsmasq.dhcphosts` file is optional. Without it, the UI still works, but reservations defined only in that file cannot be identified. Leases with a lease time of `0` are displayed as `Never` and treated as reservations.

## Building from source

Requires Go 1.25 or later. The Go source is split into several files; build the whole package, not only `main.go`.

```bash
gofmt -w *.go
go test ./...
go build -o dnsmasq-leases-ui .
```

The binary expects `templates/index.html` and the `static/` directory to be available relative to the working directory.

Run it with:

```bash
PORT=3008 ./dnsmasq-leases-ui
```

### Build the Docker image locally

The Dockerfile must copy all Go source files and build the package with `go build .` (or equivalent). Do not use `go build ./main.go`, because that compiles only the named file and omits the other source files.

```bash
docker build -t dnsmasq-leases-ui:local .
docker run --rm --network host \
  -e PORT=3008 \
  -v /var/lib/dnsmasq:/var/lib/dnsmasq:ro \
  -v ./data:/data \
  dnsmasq-leases-ui:local
```

## Docker image publishing

The current development workflow is configured for manual execution from **GitHub → Actions → Build and publish Docker image → Run workflow**. A successful run builds and pushes a multi-architecture image (`linux/amd64` and `linux/arm64`) with the tag:

```text
ghcr.io/anonymous-writer/dnsmasq-leases-ui:go-beta
```

The workflow is triggered by `workflow_dispatch` only. Publishing a GitHub Release or Pre-release does not trigger this workflow, and it does not update `latest`. To publish new development code, push all source changes to the repository and run the workflow manually.

## Source layout

The Go application is split into files by responsibility:

- `main.go` — application startup and HTTP server
- `config.go` — environment variables and runtime configuration
- `models.go` — shared data structures and normalization helpers
- `reservations.go` — parsing and matching DHCP reservations
- `leases.go` — reading and parsing the lease file
- `network.go` — ping, HTTP/HTTPS, DNS resolution, and URL checks
- `known_macs.go` — persistent remembered-MAC storage and mark/unmark/reset operations
- `handlers.go` — HTTP handlers for the UI, leases API, and remembered-device actions
- `templates/index.html` — HTML template for the web interface
- `static/app.js` — browser-side rendering, filtering, countdown, theme, and device actions

Keep the `.go` files in the same directory and in the same Go package (`package main`). The Docker build context must include all these files.

## IPv6 notes

IPv6 leases are displayed when present in the dnsmasq lease file. The app encloses IPv6 literals in brackets when used in HTTP URLs, as required by URL syntax. Detection of web interfaces requires the container to reach the relevant LAN IPv6 addresses; host networking is one option when bridge networking lacks the required IPv6 route. IPv6 entries without valid MAC addresses are not marked as new.

## Credits

- Theme-toggle icons: [Feather Icons](https://feathericons.com/) (MIT)
- Favicon: [Lucide](https://lucide.dev/) (ISC)

See [NOTICE](NOTICE) for attribution details.

## Contributing

Issues and pull requests are welcome. Please include reproduction steps for bugs and add tests where practical.

## License

MIT — see [LICENSE](LICENSE).
