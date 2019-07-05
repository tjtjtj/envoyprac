prac3
-----

staticなエンドポイント (envoy_static.yaml) と、EDS (envoy.yaml + prac3.go) をできるだけ一致させてみた。

## static な endpoint

192.168.0.32
```
docker run --rm -d -p 8080:80 dockercloud/hello-world
```

192.168.0.33
```
docker run --rm -d -p 8080:80 dockercloud/hello-world
```

192.168.0.31:app
```
curl -H 'Host: hello.local' 127.0.0.1
```

192.168.0.31:envoy
```
cp envoy_static.yaml /tmp/envoy/envoy.yaml
docker run \
    --name envoy --rm\
    --net=host \
    -v /tmp/envoy:/etc/envoy \
    envoyproxy/envoy:v1.10.0
```

192.168.0.31:app
```
curl -H 'Host: hello.local' 127.0.0.1
```

### 備考 cluster.type

`cluster.type: LOGICAL_DNS` としたところ `LOGICAL_DNS clusters must have a single locality_lb_endpoint and a single lb_endpoint` と怒られた。
STATIC or STRICT_DNS でとりあえず行けた。STATIC(default) を選択した。

```
  clusters:
  - name: hello_cluster
    connect_timeout: 0.25s
    type: LOGICAL_DNS         <--- 怒られる
```

https://www.envoyproxy.io/docs/envoy/v1.10.0/intro/arch_overview/service_discovery

- STATIC
  - 静的は最も単純なサービス検出タイプです。設定は各上流ホストの解決されたネットワーク名（IPアドレス/ポート、UNIXドメインソケットなど）を明示的に指定します。
- STRICT_DNS (厳密)
  - 厳密なDNSサービス検出を使用している場合、Envoyは指定されたDNSターゲットを継続的かつ非同期的に解決します。DNS結果に返された各IPアドレスは、アップストリームクラスタ内の明示的なホストと見なされます。
- LOGICAL_DNS (論理)
  - 論理DNSは、厳密DNSに似た非同期解決メカニズムを使用します。ただし、厳密にDNSクエリの結果を取得し、それらがアップストリームクラスタ全体を構成すると想定するのではなく、論理DNSクラスタは、新しい接続を開始する必要があるときに返される最初のIPアドレスのみを使用します。
  - 


## EDS でendpoint を得る


192.168.0.31:envoy
```
docker run \
    --name envoy --rm\
    --net=host \
    -v /tmp/envoy:/etc/envoy \
    envoyproxy/envoy:v1.10.0
```

192.168.0.31:control-plane
```
go run prac3.go
```

192.168.0.31:app
```
curl -H 'Host: hello.local' 127.0.0.1
```


### 備考 endpoint

endpoint には 3種ある。
https://www.envoyproxy.io/docs/envoy/v1.10.0/api-v2/api/v2/endpoint/endpoint.proto.htm

- endpoint.Endpoint
  - Upstream host identifier. 
  - エンドポイント1コ
- endpoint.LbEndpoint
  - An Endpoint that Envoy can route traffic to.
  - エンドポイント1コ
- endpoint.LocalityLbEndpoints
  - A group of endpoints belonging to a Locality. One can have multiple LocalityLbEndpoints for a locality, but this is generally only done if the different groups need to have different load balancing weights or different priorities.
  - エンドポイント複数持てる
