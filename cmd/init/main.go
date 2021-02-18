package main

import (
	"github.com/spf13/cobra"
	"k8s-pod-mutator-webhook/internal/logger"
	cert_generator "k8s-pod-mutator-webhook/pkg/cert-generator"
	"k8s-pod-mutator-webhook/pkg/webhook"
)

var rootCmd = &cobra.Command{
	Use:   "k8s-pod-mutator-init",
	Short: "",
	Long: `

`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.SetLogLevel(cmd.Flag("log-level").Value.String())

		initWebhook()
	},
}

var parameters = &struct {
	certOutputFiles       cert_generator.CertOutputFiles
	webhookConfigTemplate string
}{
	certOutputFiles:       cert_generator.CertOutputFiles{},
	webhookConfigTemplate: "",
}

func initWebhook() {
	webhookConfiguration, err := webhook.ConfigurationFromTemplate(parameters.webhookConfigTemplate)
	if err != nil {
		logger.Logger.Fatal(err.Error())
	}

	certs, err := cert_generator.Generate(webhookConfiguration.GetServiceMetadata(), parameters.certOutputFiles)
	if err != nil {
		logger.Logger.Fatal(err.Error())
	}

	if err = webhookConfiguration.ApplyInCluster(certs.CaCert); err != nil {
		logger.Logger.Fatal(err.Error())
	}
}

func init() {
	rootCmd.PersistentFlags().String("log-level", "info", "panic | fatal | error | warn | info | debug | trace")

	rootCmd.PersistentFlags().StringVar(&parameters.certOutputFiles.CaCertOutputFile, "ca-cert-output", "/etc/k8s-pod-mutator/certs/ca.crt", "Output file path for the CA cert.")
	rootCmd.PersistentFlags().StringVar(&parameters.certOutputFiles.CaKeyOutputFile, "ca-key-output", "/etc/k8s-pod-mutator/certs/ca.key", "Output file path for the CA key.")
	rootCmd.PersistentFlags().StringVar(&parameters.certOutputFiles.TlsCertOutputFile, "tls-cert-output", "/etc/k8s-pod-mutator/certs/tls.crt", "Output file path for the TLS cert.")
	rootCmd.PersistentFlags().StringVar(&parameters.certOutputFiles.TlsKeyOutputFile, "tls-key-output", "/etc/k8s-pod-mutator/certs/tls.key", "Output file path for the TLS key.")

	rootCmd.PersistentFlags().StringVar(&parameters.webhookConfigTemplate, "webhook-config-template", "/etc/k8s-pod-mutator/config/webhook_config_template.yaml", "Path to the manifest template file for the MutatingWebhookConfiguration")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Logger.Fatal(err.Error())
	}
}
