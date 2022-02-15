package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// client represents the wrapper of Kubernetes API client
type client struct {
	clientset *kubernetes.Clientset

	// origin is K8NSKEL_ORIGIN environment value
	origin string

	// ignoreDest is K8NSKEL_IGNORE_DEST environment value
	ignoreDests map[string]int

	// excludeSecrets contains excluding Secrets name.
	// this value is from K8NSKEL_EXCLUDE_SECRETS environment value.
	excludeSecrets map[string]struct{}
}

// newClient creates client object
func newClient(origin, ignoreDestCSV, excludeSecretsCSV string) (*client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load in-cluster config: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to load clientset: %s", err)
	}

	sv := strings.Split(ignoreDestCSV, ",")
	ignoreDests := map[string]int{
		origin: 0, // ignore origin
	}
	for _, dest := range sv {
		ignoreDests[dest] = 0 // meaningless value
	}

	es := strings.Split(excludeSecretsCSV, ",")
	excludeSecrets := make(map[string]struct{}, len(es))
	for _, secret := range es {
		excludeSecrets[secret] = struct{}{}
	}

	return &client{
		clientset:      clientset,
		origin:         origin,
		ignoreDests:    ignoreDests,
		excludeSecrets: excludeSecrets,
	}, nil
}

func (c *client) watchNamespaceEvents(ctx context.Context, wg *sync.WaitGroup) (stop bool, err error) {
	wg.Add(1)
	defer wg.Done()

	watcher, err := c.clientset.CoreV1().Namespaces().Watch(metav1.ListOptions{})
	if err != nil {
		return true, fmt.Errorf("failed to create watch interface: %s", err)
	}

	// ADDED events are also notified for namespaces that already exist at startup.
	// Therefore, necessary to ignore this notification only at startup.
	namespaces, err := c.clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return true, fmt.Errorf("failed to receive a namespace list: %s", err)
	}
	existingNS := map[string]int{}
	for _, ns := range namespaces.Items {
		existingNS[ns.ObjectMeta.Name] = 0 // meaningless value
	}

	// Event loop
	for {
		select {
		case <-ctx.Done():
			watcher.Stop()
			return true, nil
		case ev := <-watcher.ResultChan():
			if ev.Object == nil {
				// Closed because of timeout
				return false, nil
			}

			if ev.Type != watch.Added {
				continue
			}

			ns := ev.Object.(*apiv1.Namespace)

			newNS := ns.ObjectMeta.Name

			// Skip existing namaspaces only at startup
			if _, ok := existingNS[newNS]; ok {
				delete(existingNS, newNS)
				continue
			}

			secrets, err := c.clientset.CoreV1().Secrets(c.origin).List(metav1.ListOptions{})
			if err != nil {
				log.Printf("failed to receive a secret list from '%s': %s", c.origin, err)
				continue
			}

			// Copy secret from K8NSKEL_ORIGIN
			for _, secret := range secrets.Items {
				// Skip the default token
				if strings.HasPrefix(secret.Name, defaultTokenPrefix) && secret.Type == apiv1.SecretTypeServiceAccountToken {
					continue
				}

				_, err := c.clientset.CoreV1().Secrets(newNS).Create(&apiv1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secret.ObjectMeta.Name,
						Namespace: newNS,
					},
					Type: secret.Type,
					Data: secret.Data,
				})
				if err != nil {
					log.Printf("failed to copy a secret '%s' from '%s' to %s: %s", secret.ObjectMeta.Name, c.origin, newNS, err)
					continue
				}
			}
			log.Printf("all secrets were copied from '%s' to '%s'", c.origin, newNS)
		}
	}
}

func (c *client) watchSecretEvents(ctx context.Context, wg *sync.WaitGroup) (stop bool, err error) {
	wg.Add(1)
	defer wg.Done()

	watcher, err := c.clientset.CoreV1().Secrets(c.origin).Watch(metav1.ListOptions{})
	if err != nil {
		return true, fmt.Errorf("failed to create watch interface: %s", err)
	}

	// ADDED events are also notified for secrets that already exist at startup.
	// Therefore, necessary to ignore this notification only at startup.
	secrets, err := c.clientset.CoreV1().Secrets(c.origin).List(metav1.ListOptions{})
	if err != nil {
		return true, fmt.Errorf("failed to receive a secret list: %s", err)
	}
	existingSecrets := map[string]int{}
	for _, secret := range secrets.Items {
		if _, ok := c.excludeSecrets[secret.ObjectMeta.Name]; ok {
			continue
		}
		existingSecrets[secret.ObjectMeta.Name] = 0 // meaningless value
	}

	// Event loop
	for {
		select {
		case <-ctx.Done():
			watcher.Stop()
			return true, nil
		case ev := <-watcher.ResultChan():
			if ev.Object == nil {
				// Closed because of timeout
				return false, nil
			}

			secret := ev.Object.(*apiv1.Secret)

			// Skip to sync a secret when contains excluding lists
			if _, ok := c.excludeSecrets[secret.Name]; ok {
				continue
			}

			namespaces, err := c.clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
			if err != nil {
				log.Printf("failed to receive a namespace list: %s", err)
				continue
			}
			dests := []string{}
			for _, ns := range namespaces.Items {
				if _, ok := c.ignoreDests[ns.ObjectMeta.Name]; !ok {
					dests = append(dests, ns.ObjectMeta.Name)
				}
			}

			switch ev.Type {
			case watch.Added:
				// Skip existing secrets only at startup
				if _, ok := existingSecrets[secret.Name]; ok {
					delete(existingSecrets, secret.Name)
					continue
				}

				log.Printf("secret '%s' ADDED in '%s'", secret.Name, c.origin)

				for _, dest := range dests {
					_, err := c.clientset.CoreV1().Secrets(dest).Create(&apiv1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      secret.Name,
							Namespace: dest,
						},
						Type: secret.Type,
						Data: secret.Data,
					})
					if err != nil {
						log.Printf("failed to copy a secret '%s' from '%s' to '%s': %s", secret.Name, c.origin, dest, err)
						continue
					}
					log.Printf("secret '%s' was copied from '%s' to '%s'", secret.Name, c.origin, dest)
				}
			case watch.Modified:
				log.Printf("secret '%s' MODIFIED in '%s'", secret.Name, c.origin)

				for _, dest := range dests {
					_, err := c.clientset.CoreV1().Secrets(dest).Update(&apiv1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      secret.Name,
							Namespace: dest,
						},
						Type: secret.Type,
						Data: secret.Data,
					})
					if err != nil {
						log.Printf("failed to update a secret '%s' in '%s': %s", secret.Name, dest, err)
						continue
					}
					log.Printf("secret '%s' was updated in '%s'", secret.Name, dest)
				}
			case watch.Deleted:
				log.Printf("secret '%s' DELETED in'%s'", secret.Name, c.origin)

				for _, dest := range dests {
					err := c.clientset.CoreV1().Secrets(dest).Delete(secret.Name, &metav1.DeleteOptions{})
					if err != nil {
						log.Printf("failed to delete a secret '%s' in '%s': %s", secret.Name, dest, err)
						continue
					}
					log.Printf("secret '%s' was deleted in '%s'", secret.Name, dest)
				}
			}
		}
	}
}
