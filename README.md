# dnsmasq-leases-ui

[![GHCR](https://img.shields.io/badge/GHCR-docker%20image-blue?logo=github)](https://github.com/anonymous-writer/dnsmasq-leases-ui/pkgs/container/dnsmasq-leases-ui)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/anonymous-writer/dnsmasq-leases-ui)](LICENSE)

A lightweight web interface for the [dnsmasq](https://thekelleys.org.uk/dnsmasq/doc.html) DHCP leases file, written in Go. It provides a searchable, sortable view of DHCP leases and automatically detects reachable HTTP/HTTPS interfaces on client devices.

![Screenshot](https://raw.githubusercontent.com/anonymous-writer/dnsmasq-leases-ui/main/docs/screenshot.png)

## Features

- Searchable and sortable DHCP lease table
- IPv4 and IPv6 support
- Automatically detects reachable HTTP/HTTPS interfaces and makes device IP addresses clickable
- Shows a clickable hostname when it resolves to the corresponding lease IP
- Detects DHCP reservations from the optional `dnsmasq.dhcphosts` file
- Displays infinite leases as `Never`
- Dark/light theme and responsive web interface
- JSON API at `/leases`
- Bounded concurrent checks for device web interfaces
- Multi-architecture Docker image: `linux/amd64` and `linux/arm64`
- Runs as a non-root user in the provided Docker image

## Quick start

For testing the latest Go development build before it becomes stable, use the `go-beta` tag instead of `latest`.

Pull the current stable image from GitHub Container Registry:

```bash
docker pull ghcr.io/anonymous-writer/dnsmasq-leases-ui:latest
```

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
  ghcr.io/anonymous-writer/dnsmasq-leases-ui:latest
```

Open `http://<host>:3008`.

To test the development build, replace `:latest` with `:go-beta`.

Host networking is useful when the container needs to reach IPv6 devices on the LAN. With host networking, do not use `-p`/`--publish`; the application listens directly on the host network.

### Docker Compose / Podman Compose

```yaml
services:
  dnsmasq-leases-ui:
    image: ghcr.io/anonymous-writer/dnsmasq-leases-ui:latest
    container_name: dnsmasq-leases-ui
    restart: unless-stopped
    environment:
      PORT: "3008"
      HOME: /tmp
      XDG_RUNTIME_DIR: /tmp
    network_mode: host
    volumes:
      - /var/lib/dnsmasq:/var/lib/dnsmasq:ro
```

Start it with `docker compose up -d` or `podman compose up -d`.

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

## Endpoints

| Path | Description |
|---|---|
| `/` | Web interface |
| `/leases` | JSON response containing the leases array |

Example:

```bash
curl http://localhost:3008/leases
```

Each lease object includes `staticIP`, `leasetime`, `macAddress`, `ipAddress`, `name`, `webUrl`, and `webHostUrl`.

## Configuration

| Environment variable | Default | Description |
|---|---|---|
| `DNSMASQ_LEASES_FILE` | `/var/lib/dnsmasq/dnsmasq.leases` | Path to the leases file |
| `DNSMASQ_HOSTS_FILE` | `/var/lib/dnsmasq/dnsmasq.dhcphosts` | Path to the optional DHCP hosts file |
| `HOST` | `0.0.0.0` | Address the web server listens on |
| `PORT` | `5000` | Port the web server listens on |
| `WEB_UI_TIMEOUT` | `500ms` | Timeout for each HTTP/HTTPS or DNS check; accepts Go durations such as `500ms` or seconds such as `0.5` |
| `WEB_UI_MAX_WORKERS` | `16` | Maximum concurrent IP web-interface checks |
| `APP_VERSION` | `dev` | Version shown by the application |
| `APP_RELEASE_DATE` | empty | Release date shown by the application |
| `REPO_URL` | `https://github.com/anonymous-writer/dnsmasq-leases-ui` | Repository URL shown in the UI |

The `dnsmasq.dhcphosts` file is optional. Without it, the UI still works, but reservations defined only in that file cannot be identified. Leases with a lease time of `0` are displayed as `Never` and treated as reservations.

## Building from source

Requires Go 1.25 or later.

```bash
go test ./...
go build -o dnsmasq-leases-ui .
```

The binary expects `templates/index.html` and the `static/` directory to be available relative to the working directory.

Run it with:

```bash
PORT=3008 ./dnsmasq-leases-ui
```

### Build the Docker image locally

```bash
docker build -t dnsmasq-leases-ui:local .
docker run --rm --network host \
  -e PORT=3008 \
  -v /var/lib/dnsmasq:/var/lib/dnsmasq:ro \
  dnsmasq-leases-ui:local
```

## Docker image tags

- `latest` — current stable release.
- Version tags such as `1.2.0` — fixed release versions.
- `go-beta` — rolling Go test image for trying changes before they become stable.

Pre-release tags should not update `latest`.

## IPv6 notes

IPv6 detection requires the container to reach the relevant LAN IPv6 addresses. Host networking is one option when bridge networking lacks the required IPv6 route.

IPv6 literals are enclosed in brackets when used in HTTP URLs, as required by URL syntax.

## Credits

- Theme-toggle icons: [Feather Icons](https://feathericons.com/) (MIT)
- Favicon: [Lucide](https://lucide.dev/) (ISC)

See [NOTICE](NOTICE) for attribution details.

## Contributing

Issues and pull requests are welcome. Please include reproduction steps for bugs and add tests where practical.

## License

MIT — see [LICENSE](LICENSE).
