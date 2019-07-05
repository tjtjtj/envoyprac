package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"google.golang.org/grpc"
)

// NodeHash interfaceの実装。Envoyの識別子から文字列をかえすハッシュ関数を実装する。
type hash struct{}

func (hash) ID(node *core.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Cluster + "/" + node.Id
}

type upstream struct {
	Address string
	Port    uint32
}
type cluster struct {
	Name string
	Upstreams []upstream
}
type clustersInfo struct {
	Version string
	Clusters []cluster 
}

func endpoints1st() clustersInfo {
	return clustersInfo{
		Version : "0.1",
		Clusters: []cluster {
			{
				Name : "hello_cluster",
				Upstreams: []upstream {
					{Address: "192.168.0.32", Port: 8080},
					{Address: "192.168.0.33", Port: 8080},					
				},
			},
		},
	}
}
func endpoints2nd() clustersInfo {
	return clustersInfo{
		Version : "0.2",
		Clusters: []cluster {
			{
				Name : "hello_cluster",
				Upstreams: []upstream {
					{Address: "192.168.0.32", Port: 8081},
					{Address: "192.168.0.33", Port: 8081},
				},
			},
		},
	}
}

// map[string][]upstream { 
// 	"hello_cluster": {{"127.0.0.1", 8080}},
// }
// var endpoints2nd = map[string][]upstream {
// 	"hello_cluster": {{"127.0.0.1", 8081}},
// }

func createSnapshot(clinfo clustersInfo) cache.Snapshot {
	var resources []cache.Resource
	for _, cluster := range clinfo.Clusters {
		eps := make([]endpoint.LocalityLbEndpoints, len(cluster.Upstreams))
		for i, up := range cluster.Upstreams {
			eps[i] = endpoint.LocalityLbEndpoints{
				LbEndpoints: []endpoint.LbEndpoint{{
					HostIdentifier: &endpoint.LbEndpoint_Endpoint {
						Endpoint: &endpoint.Endpoint{
							Address: &core.Address{
								Address: &core.Address_SocketAddress{
									SocketAddress: &core.SocketAddress{
										Address:       up.Address,
										PortSpecifier: &core.SocketAddress_PortValue{PortValue: up.Port},
									},
								},
							},
						},
					},
				}},
			}
		}
		assignment := &api.ClusterLoadAssignment{
			ClusterName: cluster.Name,
			Endpoints:   eps,
		}
		resources = append(resources, assignment)
	}

	return cache.NewSnapshot(clinfo.Version, resources, nil, nil, nil)
}

func run(listen string, cluinfo clustersInfo) error {
	// xDSの結果をキャッシュとして設定すると、いい感じにxDS APIとして返してくれる。
	snapshotCache := cache.NewSnapshotCache(false, hash{}, nil)
	server := xds.NewServer(snapshotCache, nil)

	// NodeHashで返ってくるハッシュ値とその設定のスナップショットをキャッシュとして覚える
	err := snapshotCache.SetSnapshot("cluster.local/node0", createSnapshot(cluinfo))
	if err != nil {
		return err
	}

	// gRCPサーバーを起動してAPIを提供
	grpcServer := grpc.NewServer()
	api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)

	lsn, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}
	return grpcServer.Serve(lsn)
}

func main() {
	var listen string
	flag.StringVar(&listen, "listen", ":20000", "listen port")
	flag.Parse()

	log.Printf("Starting server with -listen=%s", listen)

	err := run(listen, endpoints1st())
	if err != nil {
		fmt.Println(os.Stderr, err)
		os.Exit(1)
	}
}