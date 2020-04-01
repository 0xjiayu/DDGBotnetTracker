# ddg botnet tracker

Track DDG.Botnet via P2P Protocol. Refer: 

1. 中文版： [DDG 升级: P2P机制加持,样本对抗增强](https://blog.netlab.360.com/dog-recent-updates-p2p-adopted-and-anti-analysis-enhanced/)
2. English: [DDG Botnet: A Frenzy of Updates before Chinese New Year](https://blog.netlab.360.com/ddg-botnet-a-frenzy-of-updates-before-chinese-new-year-2/)
3. [以 P2P 的方式追踪 DDG 僵尸网络](https://jiayu0x.com/2019/04/11/track-ddg-botnet-by-p2p-protocol/)

## 1. 说明：

现在 DDG 升级了，摒弃了使用 [Memberlist](https://github.com/0xjiayu/memberlist) 框架构建 P2P 网络的做法，转而自研简单的 P2P 协议来构建混合模式的 P2P 网络。所以，以前这套根据 Memberlist 协议特性来追踪 DDG Botnet 的 Tracker 工具目前失效了。因此我放出源代码，仅供讨论、研究，希望业界大佬多多指教。

阅读源码之前，建议先阅读以上 2 篇分析报告，有助于理解 DDG Botnet 的网络结构和恶意样本工作原理。

有问题发 issue 即可。

## 2. 关于编译、运行源码

首先需要说明的是，因为目前 DDG 的网络结构、网络协议发生了变化，这套源码已经基本失去了跟踪的功能，所以即使编译了（或者直接 `go run tracker.go` ）也不会有实际的效果。如果有朋友实在好奇，想尝试编译、运行这套源码，需要注意一些点。

### 2.1 P2P seed nodes

要加入 P2P 网络，需要至少一个活跃的 P2P Nodes 作为“介绍人”，这套源码启动时要么通过命令行参数 `-nodelist` 指定一个 `ip:port` 列表文件，要么是从 MySQL 数据库中取一批上次 Track 到的 Nodes 作为 Seed Nodes 来运行的。但是目前 DDG 的失陷主机都是运行新版 DDG 样本，不支持这套旧版的协议。只有极个别还在活跃的旧版本的失陷主机，已经没什么意义，而且我以前 Track 到的 DDG 失陷主机列表也不会放出来。

### 2.2 dependencies

项目中引用的第三方 Package 都已经在 **go.mod** 文件中指定了，所以用支持 Go Modules 的 Go 版本，在主目录下直接运行以下命令即可编译：

```
$go build -ldflags "-s -w" tracker.go
```

当然，直接 `$go run tracker.go` 也可以运行起来。

### 2.3 源码中用到的个性化常量

- **Default WAN IP**: 在 **util.go** 中，**DEFAULT_WANIP** 是手动硬编码指定的 Tracker 程序运行所在主机的 WAN IP。程序中会有通过公开 API 获取 WAN IP 的逻辑，成功获取的话，会覆盖掉这个默认值。这个默认值就是以防万一第三方 API 获取 WAN IP 失败而设置。

- **SLACK_MSG_API**: 在 **util.go** 中，Tracker 程序会将跟踪到的最新动态发送到 Slack 里，这个 API 即是  Slack App 收取信息的 Slack Hook API URL。Slack Web Hooks 官方文档：

  https://api.slack.com/messaging/webhooks
  
  Slack 推送的截图如下：

  ![](https://jiayu0x.com/imgs/botnet_tracker_slack_msg.png)

- **ROUTERIP**: 在 **tracker.go** 中，指的是Tracker 程序所在主机的 Router IP
- **MySQL DB Config**: MySQL 服务器相关的配置(Host/Port/DB Name/UserName/Password)，在 **tracker.go** 中。

### 2.4 数据库

Track 到的节点信息存储在 MySQL 数据库中，数据表结构如下：

```
+---------+----------------------+------+-----+---------+----------------+
| Field   | Type                 | Null | Key | Default | Extra          |
+---------+----------------------+------+-----+---------+----------------+
| id      | int(10) unsigned     | NO   | PRI | <null>  | auto_increment |
| ip      | char(16)             | NO   |     | <null>  |                |
| port    | smallint(5) unsigned | NO   |     | <null>  |                |
| version | smallint(5) unsigned | NO   |     | <null>  |                |
| hash    | char(32)             | YES  |     | <null>  |                |
| tdate   | datetime             | NO   |     | <null>  |                |
+---------+----------------------+------+-----+---------+----------------+
```

### 2.5 本地工作目录

默认在 `/var/` 目录下创建一个 **ddg_tracker** 工作目录，目录结构如下：

```
- ddg_tracker/
|-- cc_server.list
|-- log/
|   |-- 20190123164450.log
|   |-- 20190123180232.log
|   |-- 20190123211837.log
|   |-- 20190124000101.log
|   |-- 20190124060101.log
|   `-- ......
|-- sample/
|   |-- 104_236_156_211__8000__i_sh+8801aff2ec7c44bed9750f0659e4c533
|   |-- 104_236_156_211__8000__static__3019__fmt_i686+8c2e1719192caa4025ed978b132988d6
|   |-- 104_236_156_211__8000__static__3019__fmt_x86_64+d6187a44abacfb8f167584668e02c918
|   |-- 104_248_181_42__8000__i_sh+dc477d4810a8d3620d42a6c9f2e40b40
|   |-- 104_248_181_42__8000__static__3020__ddgs_i686+3ebe43220041fe7da8be63d7c758e1a8
|   |-- 104_248_181_42__8000__static__3020__ddgs_x86_64+d894bb2504943399f57657472e46c07d
|   |-- 104_248_251_227__8000__i_sh+55ea97d94c6d74ceefea2ab9e1de4d9f
|   |-- 104_248_251_227__8000__static__3020__ddgs_i686+3ebe43220041fe7da8be63d7c758e1a8
|   |-- 104_248_251_227__8000__static__3020__ddgs_x86_64+d894bb2504943399f57657472e46c07d
|   |-- 117_141_5_87__8000__i_sh+100d1048ee202ff6d5f3300e3e3c77cc
|   |-- 117_141_5_87__8000__i_sh+5760d5571fb745e7d9361870bc44f7a3
|   |-- 117_141_5_87__8000__static__3019__fmt_i686+8c2e1719192caa4025ed978b132988d6
|   `-- ......
`-- slave_conf/
    |-- 104_236_156_211__20190123165004.raw
    |-- 104_236_156_211__20190123185208.raw
    |-- 104_236_156_211__20190123223044.raw
    |-- 104_236_156_211__20190124012600.raw
    |-- 132_148_241_138__20190224191449.raw
    `-- ......
```

