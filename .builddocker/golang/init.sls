build-tools:
  pkg.installed:
    - pkgs:
      - build-essential

golang-1.21-go:
  archive.extracted:
    - name: /opt/go-1.21
    - source: https://golang.org/dl/go1.21.5.linux-amd64.tar.gz
    - skip_verify: True
    - enforce_toplevel: False
    - options: "--strip 1"
