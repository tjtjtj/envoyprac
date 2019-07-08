prac4
-----

Cluster discovery service


https://www.envoyproxy.io/docs/envoy/v1.10.0/configuration/overview/v2_overview


static_resources:
  listeners:
  - name: listener_0
    address:
      socket_address: { address: 0.0.0.0, port_value: 80 }
    filter_chains:
    - filters:
      - name: envoy.http_connection_manager
        config:
          stat_prefix: ingress_http
          route_config:
            name: route
            virtual_hosts:
            - name: hello_service
              domains: ["hello.local"]
              routes:
              - match: { prefix: "/" }
                route: { cluster: hello_cluster }
          http_filters:
          - name: envoy.router
  clusters:
  - name: xds_cluster
    connect_timeout: 0.25s
    lb_policy: ROUND_ROBIN
    http2_protocol_options: {}
    load_assignment:
      cluster_name: xds_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address: {address: 127.0.0.1, port_value: 20000 }

dynamic_resources:
  cds_config:
    api_config_source:
      api_type: GRPC
      grpc_services:
        envoy_grpc:
          cluster_name: xds_cluster




  - name: xds_cluster
    connect_timeout: 0.25s
    lb_policy: ROUND_ROBIN
    http2_protocol_options: {}
    load_assignment:
      cluster_name: xds_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address: {address: 127.0.0.1, port_value: 20000 }

