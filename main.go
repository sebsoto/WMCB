package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	ignitionv2 "github.com/coreos/ignition/config/v2_2"
	"github.com/vincent-petithory/dataurl"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
	kubeletConfig "k8s.io/kubelet/config/v1beta1"
)

const kubesInstallDir = "/tmp/k" //"c:\\k"

type translationFunc func([]byte) ([]byte, error)

type fileTranslation struct {
	source string
	dest   string
	translationFunc
}

func main() {
	ignitionFilePath := os.Args[1]
	ignitionFileContents, err := ioutil.ReadFile(ignitionFilePath)
	if err != nil {
		log.Fatal(err)
	}
	var filesToTranslate = []fileTranslation{
		{
			source: "/etc/kubernetes/kubelet.conf",
			dest:   "kubelet-config",
			translationFunc: func(in []byte) ([]byte, error) {
				var out []byte
				b := bufio.NewReader(bytes.NewReader(in))
				r := yaml.NewYAMLReader(b)
				doc, err := r.Read()
				if err != nil {
					return out, err
				}
				scheme := runtime.NewScheme()
				err = kubeletConfig.AddToScheme(scheme)
				if err != nil {
					return out, err
				}
				d := serializer.NewCodecFactory(scheme).UniversalDeserializer()
				config := kubeletConfig.KubeletConfiguration{}
				_, _, err = d.Decode(doc, nil, &config)
				if err != nil {
					return out, fmt.Errorf("could not decode yaml: %s\n%s", in, err)
				}
				config.CgroupDriver = "cgroupfs"
				config.Authentication.X509.ClientCAFile = filepath.Join(kubesInstallDir, "kubelet-ca.crt")
				out, err = json.Marshal(config)
				if err != nil {
					return out, err
				}
				return out, err
			},
		},
		{
			source: "/etc/kubernetes/kubeconfig",
			dest:   "bootstrap-kubeconfig",
		},
		{
			source: "/etc/kubernetes/kubelet-ca.crt",
			dest:   "kubelet-ca.crt",
		},
		{
			source: "/var/lib/kubelet/kubeconfig",
			dest:   "kubeconfig",
		},
	}
	// Parse configuration file
	configuration, _, err := ignitionv2.Parse(ignitionFileContents)
	if err != nil {
		log.Fatal(err)
	}
	for _, ignFile := range configuration.Storage.Files {
		for _, filePair := range filesToTranslate {
			if ignFile.Node.Path == filePair.source {
				log.Printf("Processing file: %s\n", ignFile.Node.Path)
				contents, err := dataurl.DecodeString(ignFile.FileEmbedded1.Contents.Source)
				if err != nil {
					log.Fatal(err)
				}
				newContents := contents.Data
				if filePair.translationFunc != nil {
					newContents, err = filePair.translationFunc(contents.Data)
					if err != nil {
						log.Fatalf("Could not process %s: %s", filePair.source, err)
					}
				}
				if err = ioutil.WriteFile(filepath.Join(kubesInstallDir, filePair.dest), newContents, 0644); err != nil {
					log.Fatalf("Could not write to %s: %s", filePair.dest, err)
				}
			}
		}
	}
}
