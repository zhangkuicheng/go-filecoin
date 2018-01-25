package libp2p

import (
	"context"
	"crypto/rand"

	bhost "gx/ipfs/QmNRN4eZGmY89CRC4T5PC4xDYRx6GkDKEfRnvrT65fVeio/go-libp2p/p2p/host/basic"
	yamux "gx/ipfs/QmNWCEvi7bPRcvqAV8AKLGVNoQdArWi7NJayka2SM4XtRe/go-smux-yamux"
	mplex "gx/ipfs/QmREBy6TSjLQMtYFhjf97cypsUTzBagcwamWocKHFCTb1e/go-smux-multiplex"
	pnet "gx/ipfs/QmRFDGFgeKQjEjZdVcDUBiGYLkRDHbH151dLwa5K7dgGZy/go-libp2p-interface-pnet"
	swarm "gx/ipfs/QmSD9fajyipwNQw3Hza2k2ifcBfbhGoC1ZHHgQBy4yqU8d/go-libp2p-swarm"
	msmux "gx/ipfs/QmVniQJkdzLZaZwzwMdd3dJTvWiJ1DQEkreVy6hs6h7Vk5/go-smux-multistream"
	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	mux "gx/ipfs/QmY9JXR3FupnYAYJWK9aMr9bCpqWKcToQ1tz8DVGTrHpHw/go-stream-muxer"
	peer "gx/ipfs/Qma7H6RW8wRrfZpNSXwxYGcd1E149s42FpWNpDNieSVrnU/go-libp2p-peer"
	crypto "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	metrics "gx/ipfs/Qmb1QrSXKwGFWgiGEcyac4s5wakJG4yPvCPk49xZHxr5ux/go-libp2p-metrics"
	transport "gx/ipfs/QmbbNoAoAymAqijje51yTMbskK1vAQivR25s1BwuLuRksf/go-libp2p-transport"
	pstore "gx/ipfs/QmeZVQzUrXqaszo24DAoHfGzcmCptN9JyngLkGAiEfk2x7/go-libp2p-peerstore"
	host "gx/ipfs/QmfCtHMCd9xFvehvHeVxtKVXJTMVTuHhyPRVHEXetn87vL/go-libp2p-host"
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
