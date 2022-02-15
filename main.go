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

	// ignoreSecretsEnvKey is key of the environment variable that contains excluding Secrets.
	// the value expects a comma-separated list of Secret names.
	excludeSecretsEnvKey = "K8NSKEL_EXCLUDE_SECRETS"

	// defaultExcludeSecrets is default value of excluding Secrets list.
	defaultExcludeSecrets = ""
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

	// Get exclude Secret lists from environment value
	excludeSecrets := defaultExcludeSecrets
	if v, ok := os.LookupEnv("K8NSKEL_EXCLUDE_SECRETS"); ok {
		excludeSecrets = v
	}

	client, err := newClient(k8nskelOrigin, k8nskelIgnoreDest, excludeSecrets)
	if err != nil {
		log.Fatalf("failed to initialize Kubernetes API client: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := new(sync.WaitGroup)

	// Start watching of namespace events
	go func() {
		for {
			stop, err := client.watchNamespaceEvents(ctx, wg)
			if err != nil {
				log.Printf("failed to start watching of namespace events: %s", err)
			}
			if stop {
				cancel()
				break
			}
		}
	}()

	// Start watching of secret events
	go func() {
		for {
			stop, err := client.watchSecretEvents(ctx, wg)
			if err != nil {
				log.Printf("failed to start watching of secret events: %s", err)
			}
			if stop {
				cancel()
				break
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	stopSignals := []os.Signal{
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}
	signal.Notify(sigCh, stopSignals...)
	go func() {
		<-sigCh
		cancel()
	}()

	wg.Wait()
}
