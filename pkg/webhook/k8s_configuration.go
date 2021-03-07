package webhook

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s-pod-mutator-webhook/internal/logger"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Configuration struct {
	template *admissionregistrationv1.MutatingWebhookConfiguration
}

func ConfigurationFromTemplate(templateFile string) (*Configuration, error) {
	logger.Logger.WithFields(logrus.Fields{
		"templateFile": templateFile,
	}).Infoln("creating k8s configuration from template")

	template, err := readAsMutatingWebhookConfiguration(templateFile)
	if err != nil {
		return nil, err
	}

	if len(template.Webhooks) != 1 {
		logger.Logger.WithFields(logrus.Fields{
			"reason":   "unexpected number of webhooks",
			"expected": "1",
			"actual":   fmt.Sprintf("%v", len(template.Webhooks)),
		}).Fatalln("invalid webhook configuration")
	}

	logger.Logger.Debugf("template: %+v", template)

	return &Configuration{template}, nil
}

func (c *Configuration) GetServiceMetadata() ServiceMetadata {
	return ServiceMetadata{
		Name:      c.template.Webhooks[0].ClientConfig.Service.Name,
		Namespace: c.template.Webhooks[0].ClientConfig.Service.Namespace,
	}
}

func (c *Configuration) ApplyInCluster(caBundle []byte) error {
	logger.Logger.WithFields(logrus.Fields{
		"name": c.template.Name,
	}).Infoln("applying k8s configuration...")

	c.template.Webhooks[0].ClientConfig.CABundle = caBundle

	client, err := createK8sClient()
	if err != nil {
		return err
	}

	logger.Logger.WithFields(logrus.Fields{
		"name": c.template.Name,
	}).Debugln("checking if k8s configuration exists...")
	existingConfig, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), c.template.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if err := c.createConfig(client); err != nil {
			return err
		}
	} else {
		if err := c.updateConfig(existingConfig, client); err != nil {
			return err
		}
	}

	logger.Logger.WithFields(logrus.Fields{
		"name": c.template.Name,
	}).Infoln("successfully applied k8s configuration")

	return nil
}

func (c *Configuration) createConfig(client *kubernetes.Clientset) error {
	logger.Logger.WithFields(logrus.Fields{
		"name": c.template.Name,
	}).Debugln("k8s configuration does not exist, creating...")

	_, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), c.template, metav1.CreateOptions{})
	return err
}

func (c *Configuration) updateConfig(existingConfig *admissionregistrationv1.MutatingWebhookConfiguration, client *kubernetes.Clientset) error {
	logger.Logger.WithFields(logrus.Fields{
		"name":            c.template.Name,
		"resourceVersion": existingConfig.ResourceVersion,
	}).Debugln("k8s configuration already exists, updating...")

	c.template.ResourceVersion = existingConfig.ResourceVersion

	_, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(context.TODO(), c.template, metav1.UpdateOptions{})
	return err
}

func readAsMutatingWebhookConfiguration(templateFile string) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	logger.Logger.WithFields(logrus.Fields{
		"templateFile": templateFile,
	}).Tracef("reading template file...")
	templateYamlBytes, err := ioutil.ReadFile(templateFile)
	if err != nil {
		return nil, err
	}

	template := &admissionregistrationv1.MutatingWebhookConfiguration{}
	err = yaml.Unmarshal(templateYamlBytes, template)
	if err != nil {
		return nil, err
	}
	return template, err
}

func createK8sClient() (*kubernetes.Clientset, error) {
	logger.Logger.Tracef("creating k8s client...")

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}
