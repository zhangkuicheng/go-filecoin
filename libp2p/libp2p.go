package libp2p

import (
	"context"
	"crypto/rand"

	yamux "gx/ipfs/QmNWCEvi7bPRcvqAV8AKLGVNoQdArWi7NJayka2SM4XtRe/go-smux-yamux"
	mplex "gx/ipfs/QmP81tTizXSjKTLWhjty1rabPQHe1YPMj6Bq5gR5fWZakn/go-smux-multiplex"
	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	transport "gx/ipfs/QmQVm7pWYKPStMeMrXNRpvAJE5rSm9ThtQoNmjNHC7sh3k/go-libp2p-transport"
	metrics "gx/ipfs/QmQbh3Rb7KM37As3vkHYnEFnzkVXNCP8EYGtHz6g2fXk14/go-libp2p-metrics"
	pnet "gx/ipfs/QmQq9YzmdFdWNTDdArueGyD7L5yyiRQigrRHJnTGkxcEjT/go-libp2p-interface-pnet"
	msmux "gx/ipfs/QmVniQJkdzLZaZwzwMdd3dJTvWiJ1DQEkreVy6hs6h7Vk5/go-smux-multistream"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	mux "gx/ipfs/QmY9JXR3FupnYAYJWK9aMr9bCpqWKcToQ1tz8DVGTrHpHw/go-stream-muxer"
	crypto "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	host "gx/ipfs/Qmc1XhrFEiSeBNn3mpfg6gEuYCt5im2gYmNVmncsvmpeAk/go-libp2p-host"
	swarm "gx/ipfs/QmdQFrFnPrKRQtpeHKjZ3cVNwxmGKKS2TvhJTuN9C9yduh/go-libp2p-swarm"
	bhost "gx/ipfs/QmefgzMbKZYsmHFkLqxgaTBG9ypeEjrdWRD5WXH4j1cWDL/go-libp2p/p2p/host/basic"
)

type Config struct {
	Transports  []transport.Transport
	Muxer       mux.Transport
	ListenAddrs []ma.Multiaddr
	PeerKey     crypto.PrivKey
	Peerstore   pstore.Peerstore
	Protector   pnet.Protector
	Reporter    metrics.Reporter
}

func Construct(ctx context.Context, cfg *Config) (host.Host, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if cfg.PeerKey == nil {
		priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
		if err != nil {
			return nil, err
		}
		cfg.PeerKey = priv
	}

	// Obtain Peer ID from public key
	pid, err := peer.IDFromPublicKey(cfg.PeerKey.GetPublic())
	if err != nil {
		return nil, err
	}

	ps := cfg.Peerstore
	if ps == nil {
		ps = pstore.NewPeerstore()
	}

	ps.AddPrivKey(pid, cfg.PeerKey)
	ps.AddPubKey(pid, cfg.PeerKey.GetPublic())

	swrm, err := swarm.NewSwarmWithProtector(ctx, cfg.ListenAddrs, pid, ps, cfg.Protector, cfg.Muxer, cfg.Reporter)
	if err != nil {
		return nil, err
	}

	netw := (*swarm.Network)(swrm)

	return bhost.New(netw), nil
}

func DefaultConfig() *Config {
	// Create a multiaddress
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
	if err != nil {
		panic(err)
	}

	// Set up stream multiplexer
	tpt := msmux.NewBlankTransport()
	tpt.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)
	tpt.AddTransport("/mplex/6.3.0", mplex.DefaultTransport)

	return &Config{
		ListenAddrs: []ma.Multiaddr{addr},
		Peerstore:   pstore.NewPeerstore(),
		Muxer:       tpt,
	}
}
