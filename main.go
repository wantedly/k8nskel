package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	// defaultK8nskelOrigin is default value of name of the namespace from which the secret is copied.
	defaultK8nskelOrigin = "k8nskel-origin"

	// defaultK8nskelIgnoreDest is default value of CSV list of namespaces that
	// does not reflect its state when secret in `K8NSKEL_ORIGIN` is added/changed/deleted.
	defaultK8nskelIgnoreDest = "kube-public,kube-system"

	// defaultTokenPrefix is prefix of default token in secret
	defaultTokenPrefix = "default-token-"
)

func main() {
	// Get origin namespace from environment value
	k8nskelOrigin := defaultK8nskelOrigin
	if v, ok := os.LookupEnv("K8NSKEL_ORIGIN"); ok {
		k8nskelOrigin = v
	}

	// Get ignore dest list from environment value
	k8nskelIgnoreDest := defaultK8nskelIgnoreDest
	if v, ok := os.LookupEnv("K8NSKEL_IGNORE_DEST"); ok {
		k8nskelIgnoreDest = v
	}

	client, err := newClient(k8nskelOrigin, k8nskelIgnoreDest)
	if err != nil {
		log.Fatalf("failed to initialize Kubernetes API client: %s\n", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := new(sync.WaitGroup)

	// Start watching of namespace events
	if err := client.watchNamespaceEvents(ctx, wg); err != nil {
		log.Fatalf("failed to start watching of namespace events: %s", err)
	}

	// Start watching of secret events
	if err := client.watchSecretEvents(ctx, wg); err != nil {
		log.Fatalf("failed to start watching of secret events: %s", err)
	}

	sigCh := make(chan os.Signal, 1)
	stopSignals := []os.Signal{
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}
	signal.Notify(sigCh, stopSignals...)

	<-sigCh
	cancel()
	wg.Wait()
}
