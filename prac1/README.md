## 実験の流れ

ここを参考（ほとんどコピーですが）にさせていただきました。 
https://i-beam.org/2019/03/13/envoy-xds-server/


- helloを起動
- envoy 起動
  - このときenvoyはhelloを知らない。 EDSのエンドポイントは知っている
- curl しても失敗するはず
- control-plane 起動
  - envoy が control-plane(EDS) に接続
  - control-plane がエンドポイントを配信
- curl するとhelloを参照するはず


## 実験

helloを起動

```
docker run --rm -d -p 8080:80 dockercloud/hello-world
```

直接 curl

```
# curl http://localhost:8080

        <h3>My hostname is 9c2075e4f35a</h3>    </body>
```

/tmp/envoy/envoy.yaml

```
略
```

envoy 起動

```
# docker run \
    --name envoy --rm --publish 80:80 \
    --net=host \
    -v /tmp/envoy:/etc/envoy \
    envoyproxy/envoy:v1.10.0
```

最初の curl は失敗

```
# curl -H 'Host: hello.local' 127.0.0.1
no healthy upstream
```

prac1.go

```
略
```

prac1.go(controle-plane) 起動後の curl は成功した。

```
# curl -H 'Host: hello.local' 127.0.0.1

        <h3>My hostname is 9c2075e4f35a</h3>    </body>
```
