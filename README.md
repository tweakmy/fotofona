# fotofona [![Build Status](https://travis-ci.com/tweakmy/fotofona.svg?branch=master)](https://travis-ci.com/tweakmy/fotofona)
DNS round robin for kube master

```./bin/fotofona -h
Use for locating the Kubernetes Master via DNS

Usage:
  fotofona [flags]
  fotofona [command]

Available Commands:
  help        Help about any command
  version     Print the version number of Fotofona

Flags:
      --alsologtostderr                  log to standard error as well as files
      --cacerts string                   verify certificates of TLS-enabled secure servers using this CA bundle for etcd
      --cert string                      identify secure client using this TLS certificate file for etcd
      --domainname string                Domain name of the kubernetes master (default "kubemaster.local")
  -h, --help                             help for fotofona
      --insecure-skip-tls-verify         skip server certificate verification for etcd
      --key string                       identify secure client using this TLS key file for etcd
      --kubeconfigpath string            enter a kubeconfig path (default "/home/tweakmy/.kube/config")
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files
      --rootpath string                  Etcd root path to store the domain (default "/skydns")
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -u, --usekubeconfig                    default to use service account; if set: use kubeconfig path 
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
  -l, --watchlabels string               watch labels for nodes to be DNS (default "node-role.kubernetes.io/master=")

Use "fotofona [command] --help" for more information about a command.
```
