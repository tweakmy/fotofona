package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {

	// kubeconfig - default kubeconfig
	var kubeconfig = filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)

	// Here you will define your flags and configuration settings.
	flagEtcdRootPath = RootCmd.PersistentFlags().StringP("rootpath", "", "/skydns", "Etcd root path to store the domain")
	flagKubeMasterDomainName = RootCmd.PersistentFlags().StringP("domainname", "", "kubemaster.local", "Domain name of the kubernetes master")
	flagKubeConfig = RootCmd.PersistentFlags().StringP("kubeconfigpath", "", kubeconfig, "enter a kubeconfig path")
	flagUseKubeConfig = RootCmd.PersistentFlags().BoolP("usekubeconfig", "u", false, "default to use service account; if set: use kubeconfig path ")
	flagWatchLabels = RootCmd.PersistentFlags().StringP("watchlabels", "l", "node-role.kubernetes.io/master=", "watch labels for nodes to be DNS")
	flaginsecureskiptlsverify = RootCmd.PersistentFlags().BoolP("insecure-skip-tls-verify", "", false, "skip server certificate verification for etcd")
	flagcacert = RootCmd.PersistentFlags().StringP("cacerts", "", "", "verify certificates of TLS-enabled secure servers using this CA bundle for etcd")
	flagcert = RootCmd.PersistentFlags().StringP("cert", "", "", "identify secure client using this TLS certificate file for etcd")
	flagkey = RootCmd.PersistentFlags().StringP("key", "", "", "identify secure client using this TLS key file for etcd")

	//Add the glog flag
	flag.Set("alsologtostderr", fmt.Sprintf("%t", true))
	flag.Set("v", "2")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	flag.CommandLine.Parse([]string{})

}

//RootCmd - Base command
var RootCmd = &cobra.Command{
	Use:   "fotofona",
	Short: "Fotofona - Kubernetes Master DNS Server for Kube client",
	Long:  `Exposed Kubernetes Master(s) via DNS`,
}

// CmdExecute - Run Cobra Main here
func CmdExecute() {
	if err := RootCmd.Execute(); err != nil {
		glog.Error("Command Error", err.Error())
		os.Exit(1)
	}
}
