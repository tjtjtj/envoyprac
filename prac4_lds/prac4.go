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
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"google.golang.org/grpc"
	types "github.com/gogo/protobuf/types"
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

/*
- name: listener_0
  address:
    socket_address: { address: 0.0.0.0, port_value: 80 }
    filter_chains:
    - filters:
      - name: envoy.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager
          stat_prefix: ingress_http
          codec_type: AUTO
          rds:
            route_config_name: local_route
            config_source:
              api_config_source:
                api_type: GRPC
                grpc_services:
                  envoy_grpc:
                    cluster_name: xds_cluster
          http_filters:
          - name: envoy.router
*/

	filter := listener.Filter{
		Name: "envoy.http_connection_manager",
		ConfigType: &listener.Filter_Config {
			Config: &types.Struct {
				Fields: map[string]*types.Value {
					"stat_prefix": &types.Value{Kind: &types.Value_StringValue{StringValue: "ingress_http"}}, 
					"codec_type": &types.Value{Kind: &types.Value_StringValue{StringValue: "AUTO"}}, 
					"rds": &types.Value{Kind: &types.Value_StructValue{StructValue: &types.Struct {
						Fields: map[string]*types.Value {
							"route_config_name": &types.Value{Kind: &types.Value_StringValue{StringValue: "local_route"}}, 
							"config_source": &types.Value{Kind: &types.Value_StructValue{StructValue: &types.Struct {
								Fields: map[string]*types.Value {
									"api_config_source": &types.Value{Kind: &types.Value_StructValue{StructValue: &types.Struct{
										Fields: map[string]*types.Value {
											"api_type": &types.Value{Kind: &types.Value_StringValue{StringValue: "GRPC"}}, 
											"grpc_services": &types.Value{Kind: &types.Value_StructValue{StructValue: &types.Struct{
												Fields: map[string]*types.Value {
													"grpc_services": &types.Value{Kind: &types.Value_StructValue{StructValue: &types.Struct{
														Fields: map[string]*types.Value {
															"envoy_grpc": &types.Value{Kind: &types.Value_StructValue{StructValue: &types.Struct{
																Fields: map[string]*types.Value {
																	"cluster_name": &types.Value{Kind: &types.Value_StringValue{StringValue: "xds_cluster"}}, 
																},
															}}},
														},
													}}},
												},
											}}},
										},
									}}}, 
								},
							}}},
						},
					}}},
					"http_filters": &types.Value{Kind: &types.Value_StructValue{StructValue: &types.Struct {
						Fields: map[string]*types.Value {
							"name": &types.Value{Kind: &types.Value_StringValue{StringValue: "envoy.router"}}, 
						},
					}}},
				},
			},
		},
	}

	filterchain := listener.FilterChain{
		Filters: []listener.Filter{filter},
	}

	lstnr := api.Listener{
		Name: "listener_0",
		Address: core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{PortValue: 80},
				},
			},
		},
		FilterChains: []listener.FilterChain{filterchain},
	}
	//listeners := []api.Listener{lstnr}

	var listenerresources []cache.Resource
	listenerresources = append(listenerresources, &lstnr)

/* RDS
	version_info: "0"
	resources:
	- "@type": type.googleapis.com/envoy.api.v2.RouteConfiguration
	  name: local_route
	  virtual_hosts:
	  - name: hello_service
		domains: ["hello.local"]
		routes:
		- match: { prefix: "/" }
		  route: { cluster: hello_cluster }
*/
	resroute := api.RouteConfiguration{
		Name: "local_route",
		VirtualHosts: []route.VirtualHost{{
			Name: "hello_service",
			Domains: []string{"hello.local"},
			Routes: []route.Route{{
				Match: route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction {
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: "hello_cluster",
						},
					},
				},
			}},
		}},
	}
	var routeResources []cache.Resource
	routeResources = append(routeResources, &resroute)


/*
	clusters:
	- name: hello_cluster
	  type: STRICT_DNS
	  connect_timeout: 0.25s
	  lb_policy: ROUND_ROBIN
	  load_assignment:
		cluster_name: hello_cluster
		endpoints:
		- lb_endpoints:
		  - endpoint:
			  address:
				socket_address: { address: hello1, port_value: 80 }
		  - endpoint:
			  address:
				socket_address: { address: hello2, port_value: 80 }
*/
	rescluster := api.Cluster {
		Name: "hello_cluster",
		ClusterDiscoveryType: &api.Cluster_Type{
			Type: api.Cluster_STRICT_DNS,
		},
		ConnectTimeout: 1,
		LbPolicy: api.Cluster_ROUND_ROBIN,
		LoadAssignment: &api.ClusterLoadAssignment{
			ClusterName: "hello_cluster",
	 		Endpoints: []endpoint.LocalityLbEndpoints{{
				LbEndpoints: []endpoint.LbEndpoint{{
					HostIdentifier: &endpoint.LbEndpoint_Endpoint {
						Endpoint: &endpoint.Endpoint{
							Address: &core.Address{
								Address: &core.Address_SocketAddress{
									SocketAddress: &core.SocketAddress{
										Address:       "192.168.0.32",
										PortSpecifier: &core.SocketAddress_PortValue{PortValue: 8080},
 									},
								},
							},
						},
	 				},
	 			}},
			}},
		},
	}


	// for _, cluster := range clinfo.Clusters {
	// 	eps := make([]endpoint.LocalityLbEndpoints, len(cluster.Upstreams))
	// 	for i, up := range cluster.Upstreams {
	// 		eps[i] = endpoint.LocalityLbEndpoints{
	// 			LbEndpoints: []endpoint.LbEndpoint{{
	// 				HostIdentifier: &endpoint.LbEndpoint_Endpoint {
	// 					Endpoint: &endpoint.Endpoint{
	// 						Address: &core.Address{
	// 							Address: &core.Address_SocketAddress{
	// 								SocketAddress: &core.SocketAddress{
	// 									Address:       up.Address,
	// 									PortSpecifier: &core.SocketAddress_PortValue{PortValue: up.Port},
	// 								},
	// 							},
	// 						},
	// 					},
	// 				},
	// 			}},
	// 		}
	// 	}
	// 	assignment := &api.ClusterLoadAssignment{
	// 		ClusterName: cluster.Name,
	// 		Endpoints:   eps,
	// 	}
	// 	resources = append(resources, assignment)
	// }

	var clusterResources []cache.Resource
	clusterResources = append(clusterResources, &rescluster)


	return cache.NewSnapshot(clinfo.Version, nil, clusterResources, routeResources, listenerresources)
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