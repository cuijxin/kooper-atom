package crd

import (
	"fmt"
	"time"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionscli "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeversion "k8s.io/apimachinery/pkg/util/version"

	"github.com/cuijxin/kooper-atom/log"
	wraptime "github.com/cuijxin/kooper-atom/wrapper/time"
)

const (
	checkCRDInterval = 2 * time.Second
	crdReadyTimeout  = 3 * time.Minute
)

var (
	clusterMinVersion = kubeversion.MustParseGeneric("v1.7.0")
	defCategories     = []string{"all", "kooper"}
)

// Scope is the scope of a CRD.
type Scope = apiextensionsv1beta1.ResourceScope

const (
	// ClusterScoped represents a type of a cluster scoped CRD.
	ClusterScoped = apiextensionsv1beta1.ClusterScoped
	// NamespaceScoped represents a type of a namespaced scoped CRD.
	NamespaceScoped = apiextensionsv1beta1.NamespaceScoped
)

// Conf is the configuration required to create a CRD
type Conf struct {
	// Kind is the kind of the CRD.
	Kind string
	// NamePlural is the plural name of the CRD (in most cases the plural of Kind).
	NamePlural string
	// ShortNames are short names of the CRD. It must be all lowercase.
	ShortNames []string
	// Group is the group of the CRD.
	Group string
	// Version is the version of the CRD.
	Version string
	// Scope is the scode of the CRD (cluster scoped or namespace scoped).
	Scope Scope
	// Categories is a way of grouping multiple resources (example `kubectl get all`),
	// Kooper adds the CRD to `all` and `kooper` categories(apart from the described in Caregories).
	Categories []string
	// EnableStatus will enable the Status subresource on the CRD. This is feature
	// entered in v1.10 with the CRD subresources.
	// By default is disabled.
	EnableStatusSubresource bool
	// EnableScaleSubresource by default will be nil and means disabled, if
	// the object is present it will set this scale configuration to the subresource.
	EnableScaleSubresource *apiextensionsv1beta1.CustomResourceSubresourceScale
}

func (c *Conf) getName() string {
	return fmt.Sprintf("%s.%s", c.NamePlural, c.Group)
}

// Interface is the CRD client that knows how to interact with k8s to manage them.
type Interface interface {
	// EnsureCreated will ensure the the CRD is present, this also means that
	// apart from creating the CRD if is not present it will wait until is
	// ready, this is a blocking operation and will return an error if timesout
	// waiting.
	EnsurePresent(conf Conf) error
	// WaitToBePresent will wait until the CRD is present, it will check if
	// is present at regular intervals until it timesout, in case of timeout
	// will return an error.
	WaitToBePresent(name string, timeout time.Duration) error
	// Delete will delete the CRD.
	Delete(name string) error
}

// Client is the CRD client implementation using API calls to kubernetes.
type Client struct {
	aeClient apiextensionscli.Interface
	logger   log.Logger
	time     wraptime.Time // Use a time wrapper so we can control the time on our tests.
}

// NewClient returns a new CRD client.
func NewClient(aeClient apiextensionscli.Interface, logger log.Logger) *Client {
	return NewCustomClient(aeClient, wraptime.Base, logger)
}

// NewCustomClient returns a new CRD client letting you set all the required parameters
func NewCustomClient(aeClient apiextensionscli.Interface, time wraptime.Time, logger log.Logger) *Client {
	return &Client{
		aeClient: aeClient,
		logger:   logger,
		time:     time,
	}
}

// EnsurePresent satisfies crd.Interface.
func (c *Client) EnsurePresent(conf Conf) error {
	if err := c.validClusterForCRDs(); err != nil {
		return err
	}

	// Get the generated name of the CRD.
	crdName := conf.getName()

	// Create subresources
	subres := c.createSubresources(conf)

	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   conf.Group,
			Version: conf.Version,
			Scope:   conf.Scope,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural:     conf.NamePlural,
				Kind:       conf.Kind,
				ShortNames: conf.ShortNames,
				Categories: c.addDefaultCategories(conf.Categories),
			},
			Subresources: subres,
		},
	}

	_, err := c.aeClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("error creating crd %s: %s", crdName, err)
		}
		return nil
	}

	c.logger.Infof("crd %s created, waiting to be ready...", crdName)
	if err := c.WaitToBePresent(crdName, crdReadyTimeout); err != nil {
		return err
	}
	c.logger.Infof("crd %s ready", crdName)

	return nil
}

// WaitToBePresent satisfies crd.Interface.
func (c *Client) WaitToBePresent(name string, timeout time.Duration) error {
	if err := c.validClusterForCRDs(); err != nil {
		return err
	}

	tout := c.time.After(timeout)
	t := c.time.NewTicker(checkCRDInterval)

	for {
		select {
		case <-t.C:
			_, err := c.aeClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(name, metav1.GetOptions{})
			// Is present, finish.
			if err == nil {
				return nil
			}
		case <-tout:
			return fmt.Errorf("timeout waiting for CRD")
		}
	}
}

// Delete satisfies crd.Interface.
func (c *Client) Delete(name string) error {
	if err := c.validClusterForCRDs(); err != nil {
		return err
	}

	return c.aeClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(name, &metav1.DeleteOptions{})
}

func (c *Client) createSubresources(conf Conf) *apiextensionsv1beta1.CustomResourceSubresources {
	if !conf.EnableStatusSubresource && conf.EnableScaleSubresource == nil {
		return nil
	}

	sr := &apiextensionsv1beta1.CustomResourceSubresources{}

	if conf.EnableStatusSubresource {
		sr.Status = &apiextensionsv1beta1.CustomResourceSubresourceStatus{}
	}

	if conf.EnableScaleSubresource != nil {
		sr.Scale = conf.EnableScaleSubresource
	}

	return sr
}

// validClusterForCRDs returns nil if cluster is ok to be used for CRDs, otherwise error.
func (c *Client) validClusterForCRDs() error {
	// Check cluster version.
	v, err := c.aeClient.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	parsedV, err := kubeversion.ParseGeneric(v.GitVersion)
	if err != nil {
		return err
	}

	if parsedV.LessThan(clusterMinVersion) {
		return fmt.Errorf("not a valid cluster version for CRDs (required >=1.7)")
	}

	return nil
}

// addDefaultCategories adds the `all` category if isn't present
func (c *Client) addDefaultCategories(categories []string) []string {
	currentCats := make(map[string]bool)
	for _, ca := range categories {
		currentCats[ca] = true
	}

	// Add default categories if required.
	for _, ca := range defCategories {
		if _, ok := currentCats[ca]; !ok {
			categories = append(categories, ca)
		}
	}

	return categories
}
