package main

// flagEtcdRootPath - root path where coreDNS should find path
var flagEtcdRootPath *string

// flagKubeMasterDNSName - root path where coreDNS should find path
var flagKubeMasterDomainName *string

// flagKubeConfig - Path to the Kubeconfig
var flagKubeConfig *string

// flagUseKubeConfig - If there is use of local kubeconfig instead of the service account
var flagUseKubeConfig *bool

// flagWatchLabels - Node Labels to be watched from k8s api server
var flagWatchLabels *string

// flaginsecureskiptlsverify -
var flaginsecureskiptlsverify *bool

// flagcacert -
var flagcacert *string

// flagcacert -
var flagcert *string

// flagcacert -
var flagkey *string

// Use for build version
var buildtimestamp string

// Use to indicate which fotofona version
var githash string
