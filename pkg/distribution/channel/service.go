package channel

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/gorp"
)

type Service struct {
	metadataDB *gorp.DB
	proxy      *leaseProxy
	resolver   *resolver
}

func New(
	cluster aspen.Cluster,
	metadataDB *gorp.DB,
	cesiumDB cesium.DB,
	transport CreateTransport,
) *Service {
	s := &Service{
		metadataDB: metadataDB,
		proxy:      newLeaseProxy(cluster, metadataDB, cesiumDB, transport),
		resolver:   &resolver{core: cluster},
	}
	return s
}

func (s *Service) NewCreate() Create { return newCreate(s.proxy) }

func (s *Service) NewRetrieve() Retrieve { return newRetrieve(s.metadataDB) }

func (s *Service) Resolve(key Key) (address.Address, error) { return s.resolver.Resolve(key) }

func (s *Service) BindResources(svc *ontology.Ontology) {
	svc.RegisterService(ontologyType, &ResourceProvider{svc: s})
	s.proxy.resources = svc
}
