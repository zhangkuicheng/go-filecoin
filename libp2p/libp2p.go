package libp2p

import (
	"context"
	"crypto/rand"

	yamux "gx/ipfs/QmNWCEvi7bPRcvqAV8AKLGVNoQdArWi7NJayka2SM4XtRe/go-smux-yamux"
	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	mplex "gx/ipfs/QmREBy6TSjLQMtYFhjf97cypsUTzBagcwamWocKHFCTb1e/go-smux-multiplex"
	host "gx/ipfs/QmRS46AyqtpJBsf1zmQdeizSDEzo1qkWR7rdEuPFAv8237/go-libp2p-host"
	bhost "gx/ipfs/QmTzs3Gp2rU3HuNayjBVG7qBgbaKWE8bgtwJ7faRxAe9UP/go-libp2p/p2p/host/basic"
	swarm "gx/ipfs/QmU219N3jn7QadVCeBUqGnAkwoXoUomrCwDuVQVuL7PB5W/go-libp2p-swarm"
	msmux "gx/ipfs/QmVniQJkdzLZaZwzwMdd3dJTvWiJ1DQEkreVy6hs6h7Vk5/go-smux-multistream"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	mux "gx/ipfs/QmY9JXR3FupnYAYJWK9aMr9bCpqWKcToQ1tz8DVGTrHpHw/go-stream-muxer"
	crypto "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	pnet "gx/ipfs/QmauYrW3kDcfZwUuqjyDCSTyaicL8tvo3a7VkAVgAEes96/go-libp2p-interface-pnet"
	metrics "gx/ipfs/QmbXmeK6KgUAkbyVGRxXknupmWAHnt6ryghT8BFSsEh2sB/go-libp2p-metrics"
	transport "gx/ipfs/Qme2XMfKbWzzYd92YvA1qnFMe3pGDR86j5BcFtx4PwdRvr/go-libp2p-transport"
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
