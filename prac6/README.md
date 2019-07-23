envoy 入門できたかな
--------------------

仕組みはわかったとはいえ動的なproxyを試していなかった。これができたら入門できたとみなせるか。

## ファイル

- envoy.yaml
  - eds, cds, rds, lds を xds から得るようにした
- prac6.go
  - ctrl+c シグナルで 設定を切り替える
  - version:0 192.168.0.12:8080 へフォワード
  - version:1 192.168.0.12:8081 へフォワード

## 検証

1コ目のhello。アドレスは192.168.0.12:8080

```
# docker run --rm -d -p 8080:80 dockercloud/hello-world
308ca1514675f0abab1ec12c9a923953270b078c19a2835b58126f662312896a
# curl localhost:8080
        <h3>My hostname is 308ca1514675</h3>    </body>
```

2コ目のhello。アドレスは192.168.0.12:8081

```
# docker run --rm -d -p 8081:80 dockercloud/hello-world
2be7dd93cf4110902e8602768a83f2f3c354048435a998ea57c0993def285059
# curl localhost:8081
        <h3>My hostname is 2be7dd93cf41</h3>    </body>
```

コントロールプレーン実行。version:0がスタート

```
# cd envoyprac/prac6
# go run prac6.go
:
2019/07/23 20:30:28 Starting server with -listen=:20000
2019/07/23 20:30:28 start grpc server version:0
```

envoy 起動

```
# cd envoyprac/prac6
# cp envoy.yaml /tmp/envoy
# docker run \
    --name envoy --rm \
    --net=host \
    -v /tmp/envoy:/etc/envoy \
    envoyproxy/envoy:v1.10.0
```

envoyが動作するnodeでcurl。1コ目のコンテナが反応している。

```
# curl -H 'Host: hello.local' 127.0.0.1
        <h3>My hostname is 308ca1514675</h3>    </body>
```

コントロールプレーンにctrl+cシグナル送信。version:1に切り替え

```
2019/07/23 20:30:28 Starting server with -listen=:20000
2019/07/23 20:30:28 start grpc server version:0
^C
2019/07/23 20:30:45 stopping grpc server...
2019/07/23 20:30:45 start grpc server version:1
```

version:1 に切り替え後のcurl。2コ目のコンテナが反応

```
# curl -H 'Host: hello.local' 127.0.0.1
        <h3>My hostname is 2be7dd93cf41</h3>    </body>
```
