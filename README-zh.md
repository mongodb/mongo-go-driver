<p align="center"><img src="etc/assets/mongo-gopher.png" width="250"></p>
<p align="center">
  <a href="https://goreportcard.com/report/go.mongodb.org/mongo-driver"><img src="https://goreportcard.com/badge/go.mongodb.org/mongo-driver"></a>
  <a href="https://godoc.org/go.mongodb.org/mongo-driver/mongo"><img src="etc/assets/godoc-mongo-blue.svg" alt="GoDoc"></a>
  <a href="https://godoc.org/go.mongodb.org/mongo-driver/bson"><img src="etc/assets/godoc-bson-blue.svg" alt="GoDoc"></a>
  <a href="https://docs.mongodb.com/ecosystem/drivers/go/"><img src="etc/assets/docs-mongodb-green.svg"></a>
</p>

# MongoDB Go Driver

支持 MongoDB 的Golang 驱动.

-------------------------
- [要求](#要求)
- [安装](#安装)
- [使用](#使用)
- [Bug和新特性报告](#Bug和新特性报告)
- [测试开发](#测试开发)
- [持续集成](#持续集成)
- [许可](#许可)

-------------------------
## 要求

- 版本 Go 1.10 或以上. 我们的目标是支持最新的Go版本.
- 版本 MongoDB 2.6  或以上.

-------------------------
## 安装

推荐开始 MongoDB Go 驱动的使用方式是使用`dep`在工程项目里安装依赖.

```bash
dep ensure -add "go.mongodb.org/mongo-driver/mongo@~1.0.0"
```

-------------------------
## 使用

开始使用驱动，导入`mongo` 包，创建一个客户端`mongo.Client`:

```go
import (
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
```

连接客户端至你运行的MongoDB的服务端:

```go
ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
err = client.Connect(ctx)
```

在单步中操作，可以使用`Connect` 函数:

```go
ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
```

调用 `Connect` 函数不会阻塞服务的发现连接。如果想了解是否已经连接一个服务，可以使用 `Ping` 方法:

```go
ctx, _ = context.WithTimeout(context.Background(), 2*time.Second)
err = client.Ping(ctx, readpref.Primary())
```

在集合中要插入一个文档，首先取得一个数据库`Database`,然后从客户端 `Client`连接中获取集合`Collection`:

```go
collection := client.Database("testing").Collection("numbers")
```

此集合 `Collection` 实体接下来可用于插入文档集:

```go
ctx, _ = context.WithTimeout(context.Background(), 5*time.Second)
res, err := collection.InsertOne(ctx, bson.M{"name": "pi", "value": 3.14159})
id := res.InsertedID
```

多个请求方法返回一个游标，此游标可像如下使用：

```go
ctx, _ = context.WithTimeout(context.Background(), 30*time.Second)
cur, err := collection.Find(ctx, bson.D{})
if err != nil { log.Fatal(err) }
defer cur.Close(ctx)
for cur.Next(ctx) {
   var result bson.M
   err := cur.Decode(&result)
   if err != nil { log.Fatal(err) }
   // 使用结果完成一些业务逻辑
}
if err := cur.Err(); err != nil {
  log.Fatal(err)
}
```

这些方法返回一个单项,此单项`SingleResult`被返回:

```go
var result struct {
    Value float64
}
filter := bson.M{"name": "pi"}
ctx, _ = context.WithTimeout(context.Background(), 5*time.Second)
err = collection.FindOne(ctx, filter).Decode(&result)
if err != nil {
    log.Fatal(err)
}
// 使用结果完成一些业务逻辑...
```

其他的例子和文档可以在例程(example)目录或在网页[on the MongoDB Documentation website](https://docs.mongodb.com/ecosystem/drivers/go/)找到.

-------------------------
## Bug和新特性报告

新的特性和漏洞/问题可以在jira: https://jira.mongodb.org/browse/GODRIVER 上提交

-------------------------
## 测试开发

此驱动程序测试可以针对多个数据库配置运行。最简单的配置是使用一个没有认证没有ssl没有解压的单例.为运行这些基本的测试，请确保单例服务运行在本地且端口27017   [`localhost:27017`],使用命令`make`(在windows上，运行`nmake`)如下:

```
TOPOLOGY=server make
```

此 `TOPOLOGY` 变量必须在运行的测试中设置。将运行"coverage、go-lint,run go-vet"并且构建这些例子

### 测试拓扑

测试多实例，设置`MONGODB_URI="<connection-string>"` 和`TOPOLOGY=replica_set` 在使用 `make` 命令时。例如：本地复本 `rs1` 由三个结点组成,分别是在端口 27017, 27018, 和 27019:

```
MONGODB_URI="mongodb://localhost:27017,localhost:27018,localhost:27018/?replicaSet=rs1" TOPOLOGY=replica_set make
```


测试多享集群，给命令 `make` 设置`MONGODB_URI="<connection-string>"` 和 `TOPOLOGY=sharded_cluster` 变量。如：共享的使用一个mongos的集群,端口都是27017

```
MONGODB_URI="mongodb://localhost:27017/" TOPOLOGY=sharder_cluster make
```

### 测试认证和SSL

测试认证和SSL,首先设置MongoDB集群的所有者和SSL 配置。测试认证需要一个用户使用`root`角色在 `admin` 数据库。Go驱动库自带的例子的证书在目录`data/certificates`。这些证书可以用来进行测试。此例命令可运行一个后台MongoDB用SSL根据测试的配置：

```
mongod \
--auth \
--sslMode requireSSL \
--sslPEMKeyFile $(pwd)/data/certificates/server.pem \
--sslCAFile $(pwd)/data/certificates/ca.pem \
--sslWeakCertificateValidation
```


运行测试使用 `make`命令,设置 `MONGO_GO_DRIVER_CA_FILE` 确定数据库使用的认证文件所在目录，设置 `MONGODB_URI` 连接服务的字符，通过设置`AUTH=auth`, 和 `SSL=ssl`.如：

```
AUTH=auth SSL=ssl MONGO_GO_DRIVER_CA_FILE=$(pwd)/data/certificates/ca.pem  MONGODB_URI="mongodb://user:password@localhost:27017/?authSource=admin" make
```

提示:
- 为使测试套件在服务端工作正常，参数标识  `--sslWeakCertificateValidation` 是必须的。
- The test suite requires the auth database to be set with `?authSource=admin`, not `/admin`.
- 测试套件需要数据库的拥有者设置为 `?authSource=admin`, 而不是 `/admin`.

### 压缩测试

MongoDB Go 驱动支持无线压缩协议通过使用`Snappy` or `zLib`.运行测试使用无线压缩协议，设置`MONGO_GO_DRIVER_COMPRESSOR` 为 `snappy` 或 `zlib`.如：

```
MONGO_GO_DRIVER_COMPRESSOR=snappy make
```

确认参数[`--networkMessageCompressors` flag](https://docs.mongodb.com/manual/reference/program/mongod/#cmdoption-mongod-networkmessagecompressors) 在服务和客户端包括 `zlib` 法测试zLib压缩。


-------------------------
## 反馈

此MongoDB Go驱动并不完善，因此，任何帮助都是欣赏的。检出[project page](https://jira.mongodb.org/browse/GODRIVER)是需要完成必要的步骤。查看我们的帮助[contribution guidelines](CONTRIBUTING.md) 获得更加详细的引导.

-------------------------
## 持续集成

提交至master分支的会自动运行在[evergreen](https://evergreen.mongodb.com/waterfall/mongo-go-driver).

-------------------------
## 感谢

<a href="https://github.com/ashleymcnamara">@ashleymcnamara</a> - Mongo Gopher Artwork

-------------------------
## 许可

MongoDB Go 驱动遵循并使用Apache[Apache License](LICENSE)协议.
