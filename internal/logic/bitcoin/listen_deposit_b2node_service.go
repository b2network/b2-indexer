package bitcoin

import (
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/tendermint/tendermint/libs/service"
	"gorm.io/gorm"
)

// ListenDepositB2NodeService sync b2node deposit event
type ListenDepositB2NodeService struct {
	service.BaseService
	db  *gorm.DB
	log log.Logger
}

// NewListenDepositB2NodeService returns a new service instance.
func NewListenDepositB2NodeService(
	db *gorm.DB,
	logger log.Logger,
) *ListenDepositB2NodeService {
	is := &ListenDepositB2NodeService{db: db, log: logger}
	return is
}
