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

The image has two build arms, selected by `BUILD_TARGET`.

Local (the default) — builds with the local Docker daemon and loads the image here:

```bash
make build    # builds docker.io/bborbe/alert-controller:<tag> locally
make buca     # builds, pushes and cleans up locally
```

`make buca`, `make upload` and `make clean` are local-only: a cluster build
pushes from inside the cluster and never loads the image on this machine.

Cluster — builds the same image as a Kubernetes Job in the `build` namespace and
pushes it to Docker Hub:

```bash
make build BUILD_TARGET=sun REF=<sha> VERSION=<tag>
```

`REF` must be the full 40-character commit SHA and must exist on `origin`;
`VERSION` must be supplied by the caller, because the cluster arm refuses to fall
back to the newest release tag. The cluster arm pushes
`docker.io/bborbe/alert-controller:<tag>`; `bborbe/nuke` then mirrors that Docker
Hub image to the private registry the deployment pulls from.

## Version

Print the build version, commit and date, then exit 0, without any other flag or
environment variable set:

```bash
alert-controller -version
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
