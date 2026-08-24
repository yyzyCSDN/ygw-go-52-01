# GraphStore 图存储与遍历引擎

GraphStore 是一个自建的内存图存储与遍历引擎：顶点与边按分片存储，边带
属性与方向；标签索引支持按属性定位入口顶点；遍历器沿边做深度受限游走，
路径结果经合并器按批次返回；变更通过 WAL 落盘并做增量索引。

## 构建与运行

```sh
go build -mod=vendor -o graphstore ./cmd/graphstore
./graphstore -addr 127.0.0.1:8612 -data ./graphstore-data
```

## HTTP 接口

- `GET /healthz` 健康与规模探测
- `GET /api/vertices` 顶点列表
- `GET /api/edges` 边列表
- `GET /api/walk?label=station&maxDepth=3` 按标签深度受限遍历
- `GET /api/walk?start=v0&maxDepth=3&pageSize=3&page=0` 分页遍历
- `POST /api/rebuild` 标签索引全量重建
- `GET /api/metrics` 指标快照
- `GET /web/browse.html` 浏览器图浏览页面

## Docker

```sh
docker build -f benzhi.Dockerfile -t ygw-go-52-01:local .
docker run -p 8612:8612 ygw-go-52-01:local
```

镜像构建使用离线 vendor 依赖，`GOPROXY=off`。
