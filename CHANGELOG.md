# v0.5.0

Released 2022-05-10

- massive overhaul to support various networks
- updates to better support graceful shutdown
- npipe support (via `winio`)
- signal handling
- windows images
- allow setting the desired hostname to use for the rewrite
- allow for disabling the hostname rewrite altogether
- allow for conditional hostname rewrites if the hostname appears to be
  non-compliant

# v0.4.2

Released 2022-04-05

- reuse existing `proxy` instance to have better resource usage both on the
  server itself and to the upstream (prevents new sessions being created on the
  upstream for each request)

# v0.4.1

Released 2022-03-15

- upgrade to `go` version `1.18`

# v0.4.0

Released 2022-03-15

- build `s390x` and `ppc64le` images

# v0.2.0

Released 2022-03-10

- wait for upstream socket before binding

# v0.1.0

Released 2022-03-09

- initial release
