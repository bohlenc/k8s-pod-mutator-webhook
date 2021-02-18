package main

import (
	"github.com/spf13/cobra"
	"k8s-pod-mutator-webhook/internal/logger"
	"k8s-pod-mutator-webhook/pkg/mutator"
	"k8s-pod-mutator-webhook/pkg/webhook"
	"os"
	"os/signal"
	"syscall"
)

var rootCmd = &cobra.Command{
	Use:   "k8s-pod-mutator-webhook",
	Short: "Kubernetes Mutating Admission Webhook for Pods. Applies arbitrary changes to Pod manifests.",
	Long: `
This webhook mutates a Pod's manifest by applying changes from a YAML file (a "patch"), which can contain virtually arbitrary changes 
- e.g. adding containers/init-containers or volumes, changing metadata etc.
After successful mutation the Pod is marked with an annotation ("k8s-pod-mutator.io/mutated=true") to prevent repeated mutation.

By default, the webhook is reachable under "https://<service_name>:8443/mutate"

For more information regarding Admission Controllers, see https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/
`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.SetLogLevel(cmd.Flag("log-level").Value.String())

		serveWebhook()
	},
}

var parameters = &struct {
	serverSettings   webhook.ServerSettings
	mutationSettings mutator.MutationSettings
}{
	serverSettings:   webhook.ServerSettings{},
	mutationSettings: mutator.MutationSettings{},
}

func serveWebhook() {
	mutator, err := mutator.CreateMutator(parameters.mutationSettings)
	if err != nil {
		logger.Logger.Fatal(err.Error())
	}

	server, err := webhook.CreateServer(parameters.serverSettings, *mutator)
	if err != nil {
		logger.Logger.Fatal(err.Error())
	}

	go func() {
		if err := server.Start(); err != nil {
			logger.Logger.Fatal(err.Error())
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	_ = server.Stop()
}

func init() {
	rootCmd.PersistentFlags().String("log-level", "info", "panic | fatal | error | warn | info | debug | trace")

	rootCmd.PersistentFlags().IntVar(&parameters.serverSettings.Port, "port", 8443, "Port to listen on for HTTP requests.")
	rootCmd.PersistentFlags().BoolVar(&parameters.serverSettings.Tls, "tls", true, "Enables/Disables TLS.")
	rootCmd.PersistentFlags().StringVar(&parameters.serverSettings.TlsCertFile, "tls-cert", "/etc/k8s-pod-mutator/certs/tls.crt", "Path to TLS cert. Has no effect when '--tls=false'.")
	rootCmd.PersistentFlags().StringVar(&parameters.serverSettings.TlsKeyFile, "tls-key", "/etc/k8s-pod-mutator/certs/tls.key", "Path to TLS key. Has no effect when '--tls=false.'")

	rootCmd.PersistentFlags().StringVar(&parameters.mutationSettings.PatchFile, "patch", "/etc/k8s-pod-mutator/config/patch.yaml", "Path to the YAML file containing the patch to be applied to eligible Pods (see https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#pod-v1-core for help).")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Logger.Fatal(err.Error())
	}
}
