package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"sort"

	"github.com/petebowden/edge-deploy/apis/edge/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type manifest struct {
	PodSpec *v1alpha1.InternalPodspec
	os.FileInfo
}

func main() {

	nodeName := flag.String("nodename", "testedgenode", "Name of the edge node")
	podDirectory := flag.String("k8s manifest directory", "./etc/kubernetes/manifests/", "Directory for kuberenetes manifests")

	flag.Parse()

	if *nodeName == "" {
		fmt.Printf("missing nodename field")
		os.Exit(2)
	}

	config := &rest.Config{
		Host: "https://api.ocp.lab.rastapopulous.com:6443/",
		//Username:    "agent",
		BearerToken: "eyJhbGciOiJSUzI1NiIsImtpZCI6ImUzeUdZU0RteWJGeURUVGpWcVNBdTFkN3Ric3JYZTl6MUZKLThuRTR3eEEifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJvcGVuc2hpZnQtY29uZmlnIiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZWNyZXQubmFtZSI6ImFnZW50LXRva2VuLXNwem02Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZXJ2aWNlLWFjY291bnQubmFtZSI6ImFnZW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZXJ2aWNlLWFjY291bnQudWlkIjoiOGRlZmVhZjYtYThmMy00MjJhLWIzMTAtYTA2ZDExYjYzYTVkIiwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50Om9wZW5zaGlmdC1jb25maWc6YWdlbnQifQ.JsXq7leensztNNQt2VRodGD-WbixI7jqJaGeEcI1hO26N9rk-6ZUKGfksDJWwR1wmZkfHBulITXYUL2F4oXcmjin9Ms12ZuMwXoYO5Nmuuv9UgslfxN1kf5Ee53604nWU33frAtpzuklyhmaI204QDA1SDw0IOWA7--Dbgp8NiMxnvRqZPuiL5xhYx9ZDUR0DWmCdSh50hzP8H_No1wMRc8gG2qYhi9CscM4zaYjV8zA_EunWHszrXNCw8ZuJA2nPRQy4kLq8WOIdENoru_aBDETfArGcGGRFkR0Mo7OSarN_IA9-rUCSymkvxgFDorEFrizatZk2ANcZNM-_SP-Dw",
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}

	err := v1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		fmt.Println("Failed to register scheme: " + err.Error())
		os.Exit(1)
	}
	clientOptions := &client.Options{
		Scheme: scheme.Scheme,
	}
	cl, err := client.New(config, *clientOptions)
	if err != nil {
		fmt.Println("failed to create client: " + err.Error())
		os.Exit(1)
	}

	desiredEdgePodSpecList := &v1alpha1.EdgePodList{}

	// TODO: remove hardcoded namespace
	// TODO: Search based on nodename
	err = cl.List(context.Background(), desiredEdgePodSpecList, client.InNamespace("openshift-config"))
	if err != nil {
		fmt.Printf("failed to list pods in namespace default: %v\n", err)
		os.Exit(1)
	}

	// Get all files in directory

	files, err := ioutil.ReadDir(*podDirectory)
	if err != nil {
		fmt.Printf("Failed to read directory: " + *podDirectory + " Has it been mounted?")
		os.Exit(1)
	}

	currentManifests := []manifest{}

	for _, fileInfo := range files {
		byt, _ := ioutil.ReadFile(fileInfo.Name())
		podSpec := &v1alpha1.InternalPodspec{}
		if err := json.Unmarshal(byt, podSpec); err != nil {
			fmt.Printf("Failed to read file: " + fileInfo.Name())
		}
		manifest := manifest{
			PodSpec:  podSpec,
			FileInfo: fileInfo,
		}
		currentManifests = append(currentManifests, manifest)
	}

	// Sort the lists
	sort.Slice(desiredEdgePodSpecList.Items, func(i, j int) bool {
		return desiredEdgePodSpecList.Items[i].Podspec.ObjectMeta.Name > desiredEdgePodSpecList.Items[j].Podspec.ObjectMeta.Name
	})

	sort.Slice(currentManifests, func(i, j int) bool {
		return currentManifests[i].PodSpec.ObjectMeta.Name > currentManifests[j].PodSpec.ObjectMeta.Name
	})

	i, j := 0, 0
	// Compare the lists, updating or deleting as needed
	for i < len(desiredEdgePodSpecList.Items) && j < len(currentManifests) {

		desiredEdgePodName := desiredEdgePodSpecList.Items[i].Podspec.ObjectMeta.Name
		currentEdgePodName := currentManifests[j].PodSpec.ObjectMeta.Name
		if desiredEdgePodName == currentEdgePodName {

			// Does EdgePod Spec match?
			if !reflect.DeepEqual(desiredEdgePodSpecList.Items[j], currentManifests[i]) {
				// Doesn't match update file
				writePodSpec(desiredEdgePodSpecList.Items[j].Podspec, currentManifests[i].FileInfo.Name(), *podDirectory)
			}
			//increment both pointers
			i, j = i+1, j+1

		} else if desiredEdgePodName < currentEdgePodName {
			// Do we need to delete a PodSpec on disk?
			// Yes
			deletePodSpec(currentManifests[j].FileInfo.Name(), *podDirectory)
			j++
		} else {
			// No, create a new PodSpec
			writePodSpec(desiredEdgePodSpecList.Items[i].Podspec, desiredEdgePodName, *podDirectory)
			i++
		}
	}

	// Create remaining PodSpec
	for ; i < len(desiredEdgePodSpecList.Items); i++ {
		writePodSpec(desiredEdgePodSpecList.Items[i].Podspec,
			desiredEdgePodSpecList.Items[i].Podspec.ObjectMeta.Name, *podDirectory)
	}

	// Delete remaining uneeded PodSpecs
	for ; j < len(currentManifests); j++ {
		deletePodSpec(currentManifests[j].FileInfo.Name(), *podDirectory)
		j++
	}

}

func deletePodSpec(filename string, directory string) {
	err := os.Remove(directory + filename)
	if err != nil {
		fmt.Printf("Failed to delete podspec file: " + filename + " Error: " + err.Error())
	}
}

func writePodSpec(podSpec *v1alpha1.InternalPodspec, filename string, directory string) {
	byt, err := json.Marshal(podSpec)
	if err != nil {
		fmt.Printf("Failed to marshal podspec: " + podSpec.Name + " error: " + err.Error())
	}
	err = ioutil.WriteFile(directory+filename, byt, 0644)
	if err != nil {
		fmt.Printf("Failed to write podspec: " + podSpec.Name + "to file: " + filename + " error: " + err.Error())

	}
}
