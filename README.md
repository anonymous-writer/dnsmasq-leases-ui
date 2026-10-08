# dnsmasq-leases-ui — Go prototype

Go reimplementation of the current Python backend. It keeps the JSON field names and core UI behaviour: lease parsing, DHCP reservations, HTTP/HTTPS detection, hostname links, search, sorting, dark mode and periodic refresh.

Defaults:
- `/var/lib/dnsmasq/dnsmasq.leases`
- `/var/lib/dnsmasq/dnsmasq.dhcphosts`

Override with `DNSMASQ_LEASES_FILE` and `DNSMASQ_HOSTS_FILE`.

Run locally:
```bash
go run .
```

The production container uses one Go binary; Flask and Gunicorn are no longer needed.
