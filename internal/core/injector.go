package core

import (
	"github.com/arewedaks/zen-go-box/internal/config"
)

// Injector interface mendefinisikan contract untuk memodifikasi konfigurasi kernel proxy core
type Injector interface {
	Prepare(cfg *config.Config) error
}
