package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func init() {

	// kubeconfig - default kubeconfig
	var kubeconfig = filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)

	// Here you will define your flags and configuration settings.
	flagKubeConfig = RootCmd.PersistentFlags().StringP("kubeconfigpath", "", kubeconfig, "enter a kubeconfig path")
	flagUseKubeConfig = RootCmd.PersistentFlags().BoolP("usekubeconfig", "u", false, "default to use service account; if set: use kubeconfig path ")
	flagWatchLabels = RootCmd.PersistentFlags().StringP("watchlabels", "l", "node-role.kubernetes.io/master=", "watch labels for nodes to be DNS")
	flaginsecureskiptlsverify = RootCmd.PersistentFlags().BoolP("insecure-skip-tls-verify", "", false, "skip server certificate verification for etcd")
	flagcacert = RootCmd.PersistentFlags().StringP("cacerts", "", "", "verify certificates of TLS-enabled secure servers using this CA bundle for etcd")
	flagcert = RootCmd.PersistentFlags().StringP("cert", "", "", "identify secure client using this TLS certificate file for etcd")
	flagkey = RootCmd.PersistentFlags().StringP("key", "", "", "identify secure client using this TLS key file for etcd")
}

//RootCmd - Base command
var RootCmd = &cobra.Command{
	Use:   "fotofona",
	Short: "Fotofona - Kubernetes Node DNS Server for client",
	Long:  `???`,
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println(*flagKubeConfig)
		fmt.Println(*flagWatchLabels)
		fmt.Println(*flagUseKubeConfig)
	},
}

// CmdExecute - Run Cobra Main here
func CmdExecute() {
	if err := RootCmd.Execute(); err != nil {
		glog.Error("Command Error", err.Error())
		os.Exit(1)
	}
}
