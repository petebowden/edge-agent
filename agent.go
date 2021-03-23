package main

import (
	"context"
	"fmt"
	"os"

	"github.com/petebowden/edge-deploy/apis/edge/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
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

	podList := &v1alpha1.EdgePodList{}

	err = cl.List(context.Background(), podList, client.InNamespace("openshift-config"))
	if err != nil {
		fmt.Printf("failed to list pods in namespace default: %v\n", err)
		os.Exit(1)
	}

}
