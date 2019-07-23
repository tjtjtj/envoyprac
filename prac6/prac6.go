package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoyendpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	envoylistener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	envoyroute "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoyhcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache"
	envoyserver "github.com/envoyproxy/go-control-plane/pkg/server"
	envoyutil "github.com/envoyproxy/go-control-plane/pkg/util"
	grpc "google.golang.org/grpc"
)

type hash struct{}

func (hash) ID(node *envoycore.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Cluster + "/" + node.Id
}

func createListener() *envoyapi.Listener {
	/*
		https://www.envoyproxy.io/docs/envoy/v1.10.0/configuration/overview/v2_overview#example
		version_info: "0"
		resources:
		- "@type": type.googleapis.com/envoy.api.v2.Listener
		  name: listener_0
		  address:
		    socket_address:
		      address: 0.0.0.0
		      port_value: 8000
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
	manager := &envoyhcm.HttpConnectionManager{
		StatPrefix: "http",
		RouteSpecifier: &envoyhcm.HttpConnectionManager_Rds{
			Rds: &envoyhcm.Rds{
				RouteConfigName: "hello_route",
				ConfigSource: envoycore.ConfigSource{
					ConfigSourceSpecifier: &envoycore.ConfigSource_ApiConfigSource{
						ApiConfigSource: &envoycore.ApiConfigSource{
							ApiType: envoycore.ApiConfigSource_GRPC,
							GrpcServices: []*envoycore.GrpcService{{
								TargetSpecifier: &envoycore.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &envoycore.GrpcService_EnvoyGrpc{
										ClusterName: "xds_cluster",
									},
								},
							}},
						},
					},
				},
			},
		},
		HttpFilters: []*envoyhcm.HttpFilter{{
			Name: "envoy.router",
		}},
	}
	filterConfig, err := envoyutil.MessageToStruct(manager)
	if err != nil {
		panic(err.Error())
	}
	listener := &envoyapi.Listener{
		Name: "listener_0",
		Address: envoycore.Address{
			Address: &envoycore.Address_SocketAddress{
				SocketAddress: &envoycore.SocketAddress{
					Address:       "0.0.0.0",
					PortSpecifier: &envoycore.SocketAddress_PortValue{PortValue: 80},
				},
			},
		},
		FilterChains: []envoylistener.FilterChain{{
			Filters: []envoylistener.Filter{{
				Name: "envoy.http_connection_manager",
				ConfigType: &envoylistener.Filter_Config{
					Config: filterConfig,
				},
			}},
		}},
	}
	return listener
}

func createRouteConfig() *envoyapi.RouteConfiguration {
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
	routeconfig := &envoyapi.RouteConfiguration{
		Name: "hello_route",
		VirtualHosts: []envoyroute.VirtualHost{{
			Name:    "hello_service",
			Domains: []string{"hello.local"},
			Routes: []envoyroute.Route{{
				Match: envoyroute.RouteMatch{
					PathSpecifier: &envoyroute.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &envoyroute.Route_Route{
					Route: &envoyroute.RouteAction{
						ClusterSpecifier: &envoyroute.RouteAction_Cluster{
							Cluster: "hello_cluster",
						},
					},
				},
			}},
		}},
	}
	return routeconfig
}

func createCluster() *envoyapi.Cluster {
	/*
		version_info: "0"
		resources:
		- "@type": type.googleapis.com/envoy.api.v2.Cluster
		  name: hello_cluster
		  connect_timeout: 0.25s
		  lb_policy: ROUND_ROBIN
		  type: EDS
		  eds_cluster_config:
		    eds_config:
		      api_config_source:
		        api_type: GRPC
		        grpc_services:
		          envoy_grpc:
					cluster_name: xds_cluster
	*/
	connectionTimeout := time.Duration(60*1000) * time.Millisecond

	cluster := &envoyapi.Cluster{
		Name:           "hello_cluster",
		ConnectTimeout: connectionTimeout,
		LbPolicy:       envoyapi.Cluster_ROUND_ROBIN,
		ClusterDiscoveryType: &envoyapi.Cluster_Type{
			Type: envoyapi.Cluster_EDS,
		},
		EdsClusterConfig: &envoyapi.Cluster_EdsClusterConfig{
			EdsConfig: &envoycore.ConfigSource{
				ConfigSourceSpecifier: &envoycore.ConfigSource_ApiConfigSource{
					ApiConfigSource: &envoycore.ApiConfigSource{
						ApiType: envoycore.ApiConfigSource_GRPC,
						GrpcServices: []*envoycore.GrpcService{{
							TargetSpecifier: &envoycore.GrpcService_EnvoyGrpc_{
								EnvoyGrpc: &envoycore.GrpcService_EnvoyGrpc{
									ClusterName: "xds_cluster",
								},
							},
						}},
					},
				},
			},
		},
	}
	return cluster
}

func createEndpoint(endp MyEndpoint) *envoyapi.ClusterLoadAssignment {
	/*
		version_info: "0"
		resources:
		- "@type": type.googleapis.com/envoy.api.v2.ClusterLoadAssignment
		  cluster_name: hello_cluster
		  endpoints:
		  - lb_endpoints:
		    - endpoint:
		        address:
		          socket_address:
		            address: 192.186.0.32
					port_value: 80
	*/
	clusterLoadAssignment := &envoyapi.ClusterLoadAssignment{
		ClusterName: endp.name,
		Endpoints: []envoyendpoint.LocalityLbEndpoints{{
			LbEndpoints: []envoyendpoint.LbEndpoint{{
				HostIdentifier: &envoyendpoint.LbEndpoint_Endpoint{
					Endpoint: &envoyendpoint.Endpoint{
						Address: &envoycore.Address{
							Address: &envoycore.Address_SocketAddress{
								SocketAddress: &envoycore.SocketAddress{
									Address:       endp.address,
									PortSpecifier: &envoycore.SocketAddress_PortValue{PortValue: endp.port},
								},
							},
						},
					},
				},
			}},
		}},
	}
	return clusterLoadAssignment
}

func createSnapshot(endp MyEndpoint) envoycache.Snapshot {

	var endpoints []envoycache.Resource
	endpoints = append(endpoints, createEndpoint(endp))
	var clusters []envoycache.Resource
	clusters = append(clusters, createCluster())
	var routes []envoycache.Resource
	routes = append(routes, createRouteConfig())
	var listeners []envoycache.Resource
	listeners = append(listeners, createListener())

	return envoycache.NewSnapshot(endp.version, endpoints, clusters, routes, listeners)
}

func run(listen string, endp MyEndpoint) error {
	// xDSの結果をキャッシュとして設定すると、いい感じにxDS APIとして返してくれる。
	snapshotCache := envoycache.NewSnapshotCache(false, hash{}, nil)
	server := envoyserver.NewServer(snapshotCache, nil)

	// NodeHashで返ってくるハッシュ値とその設定のスナップショットをキャッシュとして覚える
	err := snapshotCache.SetSnapshot("cluster.local/node0", createSnapshot(endp))
	if err != nil {
		return err
	}

	// gRCPサーバーを起動してAPIを提供
	grpcServer := grpc.NewServer()
	envoyapi.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	envoyapi.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	envoyapi.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	envoyapi.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	lsn, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}
    go func() {
        log.Printf("start grpc server version:%s", endp.version)
        grpcServer.Serve(lsn)
    }()

    quit := make(chan os.Signal)
    signal.Notify(quit, os.Interrupt)
    <-quit
	log.Println("stopping grpc server...")

	grpcServer.Stop()
	// GracefulStop だと envoy が離してくてない
	//grpcServer.GracefulStop()

	return nil
}

func main() {
	var listen string
	flag.StringVar(&listen, "listen", ":20000", "listen port")
	flag.Parse()

	log.Printf("Starting server with -listen=%s", listen)

	end0 := MyEndpoint{
		version: "0",
		name: "hello_cluster",
		address: "192.168.0.12",
		port: 8080,
	}
	err := run(listen, end0)
	if err != nil {
		fmt.Println(os.Stderr, err)
		os.Exit(1)
	}

	end1 := MyEndpoint{
		version: "1",
		name: "hello_cluster",
		address: "192.168.0.12",
		port: 8081,
	}
	err = run(listen, end1)
	if err != nil {
		fmt.Println(os.Stderr, err)
		os.Exit(1)
	}
}

type MyEndpoint struct {
	version string
	name string
	address string
	port uint32
}
