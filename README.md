# alert-controller

A small Kubernetes controller that reloads a Prometheus (or compatible) alerting backend from
custom `Alert` resources, exposing a tiny HTTP admin surface to trigger reloads and adjust log level
at runtime.

## Run locally

```bash
make test
make run
```

## Build & publish image

```bash
make buca   # builds + pushes docker.io/bborbe/alert-controller:<git-tag>
```

## HTTP admin endpoints

Served on the configured listen address (`-listen`):

```
GET  /setloglevel/{n}   # set glog verbosity to n
GET  /trigger           # run a reload cycle
GET  /trigger/force     # force a reload cycle
GET  /reload            # reload configuration
GET  /list              # list known alerts
GET  /rules             # dump current rules
```

## License

BSD-2-Clause — see [LICENSE](LICENSE).
