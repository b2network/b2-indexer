package particle_test

import (
	"os"
	"testing"

	"github.com/b2network/b2-indexer/pkg/particle"
	"github.com/stretchr/testify/require"
)

func mockParticle(t *testing.T) *particle.Particle {
	projectID := os.Getenv("BITCOIN_BRIDGE_AA_PARTICLE_PROJECT_ID")
	serverKey := os.Getenv("BITCOIN_BRIDGE_AA_PARTICLE_SERVER_KEY")
	p, err := particle.NewParticle("https://rpc.particle.network/evm-chain",
		projectID,
		serverKey, 1102)
	require.NoError(t, err)
	return p
}

func TestNewParticle(t *testing.T) {
	type args struct {
		rpc       string
		projectID string
		serverKey string
		chanID    int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				rpc:       "http://127.0.0.1/aaa",
				projectID: "projectID",
				serverKey: "serverKey",
				chanID:    111,
			},
			wantErr: false,
		},
		{
			name: "api url fail",
			args: args{
				rpc:       "127.0.0.1.2/aaa",
				projectID: "projectID",
				serverKey: "serverKey",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := particle.NewParticle(tt.args.rpc, tt.args.projectID, tt.args.serverKey, tt.args.chanID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewParticle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestParticle_AAGetBTCAccount(t *testing.T) {
	tests := []struct {
		name      string
		btcPubKey []string
		wantErr   bool
	}{
		{
			name:      "success",
			btcPubKey: []string{"03c71750091f3f7d078f81f8b798cce98d424768dff8155b7643f7387bafd11edf"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := mockParticle(t)
			_, err := p.AAGetBTCAccount(tt.btcPubKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("AAGetBTCAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
