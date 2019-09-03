package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	ignitionv2 "github.com/coreos/ignition/config/v2_2"
	"github.com/vincent-petithory/dataurl"
)

const kubernetesInstallDirectory = "/tmp/k" //"c:\\k"

func main() {
	ignitionFilePath := os.Args[1]
	ignitionFileContents, err := ioutil.ReadFile(ignitionFilePath)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}
	var sourceDestPairs = map[string]string{
		"/etc/kubernetes/kubelet.conf":   filepath.Join(kubernetesInstallDirectory, "kubelet-config"),
		"/etc/kubernetes/kubeconfig":     filepath.Join(kubernetesInstallDirectory, "bootstrap-kubeconfig"),
		"/etc/kubernetes/kubelet-ca.crt": filepath.Join(kubernetesInstallDirectory, "kubelet-ca.crt"),
		"/var/lib/kubelet/kubeconfig":    filepath.Join(kubernetesInstallDirectory, "kubeconfig"),
	}
	// Parse configuration file
	configuration, _, err := ignitionv2.Parse(ignitionFileContents)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}
	for _, ignFile := range configuration.Storage.Files {
		for src, dest := range sourceDestPairs {
			if ignFile.Node.Path == src {
				fmt.Printf("File: %s\n", ignFile.Node.Path)
				contents, err := dataurl.DecodeString(ignFile.FileEmbedded1.Contents.Source)
				if err != nil {
					fmt.Printf("%s\n", err)
					return
				}
				//fmt.Printf("%s\n", contents.Data)
				err = ioutil.WriteFile(dest, contents.Data, 0644)
				if err != nil {
					fmt.Printf("Could not write %s to %s", src, dest)
				}
			}
		}
	}
}
