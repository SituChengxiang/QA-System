# 插件开发相关

- [x] 开源替换 TTLCache,降低项目的复杂程度
- [x] 开源替换 gopool,降低项目的复杂程度
- [x] 多用面向对象的方法,例如manager
- [ ] 插件不需要以goroutine的方式一直存在,在异步消费的时候再去执行就行了
- [ ] 二进制文件 .so 和 .dll 的真动态加载
- [ ] 插件生命周期管控
- [ ] 更好的微架构融入
- [ ] 性能优化

## v0.0.1  

基本想法：internal/pkg/extension包负责所有插件的管理、调度  
把所有插件作为 plugins 包的一部分，在main函数启动时隐性导入。通过每个插件自己的init函数，所有的插件将自己注册到manager提供的管理器内部。

manager再根据配置文件里写的顺序依次加载、调用插件

### 基本思想

1. **插件接口**：定义一个标准的插件接口，确保所有插件都遵循相同的规范。
2. **插件注册**：通过 `init` 函数在程序启动时自动注册插件。
3. **配置管理**：使用配置文件决定哪些插件需要加载及其顺序。
4. **插件执行**：在主程序中调用插件管理器来加载和执行插件。

### 步骤概述

1. **定义插件接口**
2. **编写具体插件**
3. **配置文件管理**

---

### 1. 定义插件接口

首先，我们查看定义插件接口 `Plugin` 和元数据结构 `PluginMetadata`。这些定义放在 `pkg/extension/interface.go` 文件中。你可以直接参考实例插件plugin1和plugin2来着。

```go
// pkg/extension/interface.go
package extension

// PluginMetadata 插件元数据
type PluginMetadata struct {
    Name        string // 插件名称
    Version     string // 插件版本
    Author      string // 插件作者
    Description string // 插件描述
}

// Plugin 定义插件接口
type Plugin interface {
    GetMetadata() PluginMetadata                 // GetMetadata 获取插件元数据
    Execute() error // Execute 执行插件具体的功能
}
```

### 2. 编写具体插件

每个插件都需要实现 `extension.Plugin` 接口，并通过 `init` 函数在程序启动时注册自己。我们直接以老的`emailNotifier`插件为例。（这个插件现在不会被加载，因为他被.disabled掉了，跟HMCL学的）。

示例插件: `emailNotifier.go`：（用redis stream做的消息传递，所以会有相关配置，现在已经没了）

```go
// plugins/emailNotifier.go
package plugins
// 所有的插件都属于plugin包

// EmailNotifier 插件需要的基本信息
type EmailNotifier struct {
 smtpHost     string                // SMTP服务器地址
 smtpPort     int                   // SMTP服务器端口
 smtpUsername string                // SMTP服务器用户名
 smtpPassword string                // SMTP服务器密码
 from         string                // 发件人地址
 streamName   string                // stream的名称
 groupName    string                // stream的消费者组名称
 consumerOld  string                // 处理pending消息的消费者
 consumerNew  string                // 处理新消息的消费者
 workerNum    int                   // 工作协程数量
 jobChan      chan redisv9.XMessage // 任务通道
}

// init 注册插件
func init() {
 notifier := &EmailNotifier{
  consumerOld: "consumerOld",
  consumerNew: "consumerNew",
 }
 if err := notifier.initialize(); err != nil {
  panic(fmt.Sprintf("Failed to initialize emailnotifier: %v", err))
 }
 extension.GetDefaultManager().RegisterPlugin(notifier)
}

// GetMetadata 所有插件都需要的返回插件的元数据
func (p *EmailNotifier) GetMetadata() extension.PluginMetadata {
 _ = p
 return extension.PluginMetadata{
  Name:        "emailNotifier",
  Version:     "0.1.0",
  Author:      "SituChengxiang, Copilot, Qwen2.5, DeepSeek",
  Description: "Send email notifications for new survey responses",
 }
}

// Execute 启动消费者，从这个函数出发，每个插件调用执行自己用来实现各种功能的函数
func (p *EmailNotifier) Execute() error {
 fmt.Println("Another version of the email notifier has been released, you can change to that one as this one relies on redis stream")
 ctx := context.Background()
 zap.L().Info("Email notifier started", zap.Int("workers", p.workerNum))
}

// initialize 从配置文件中读取配置信息
func (p *EmailNotifier) initialize() error {
 // 读取SMTP配置
 p.smtpHost = config.Config.GetString("emailnotifier.smtp.host")
 p.smtpPort = config.Config.GetInt("emailnotifier.smtp.port")
 p.smtpUsername = config.Config.GetString("emailnotifier.smtp.username")
 p.smtpPassword = config.Config.GetString("emailnotifier.smtp.password")
 p.from = config.Config.GetString("emailnotifier.smtp.from")

 if p.smtpHost == "" || p.smtpUsername == "" || p.smtpPassword == "" || p.from == "" {
  return errors.New("invalid SMTP configuration, this may lead to email sending failure")
 }

 // 读取Stream配置，老版本的插件是靠 redis stream做的通知提醒
 p.streamName = config.Config.GetString("redis.stream.name")
 p.groupName = config.Config.GetString("redis.stream.group")

 if p.streamName == "" || p.groupName == "" {
  zap.L().Warn("Stream name or group name is empty, email notifier will not work")
 }

 // 读取工作协程配置
 p.workerNum = config.Config.GetInt("emailnotifier.worker.num")
 if p.workerNum <= 0 {
  p.workerNum = 3 // 默认3个工作协程
 }
 p.jobChan = make(chan redisv9.XMessage, p.workerNum*2)
 return nil
}
// 具体的工作流程……

```

### 3. 配置文件管理

编辑conf目录下的配置文件 `config.yaml` :

```yaml
# conf/config.yaml
plugins:
  order:
    - "plugin1"
    - "plugin2"
    - "emailNotifier"
```

**一定要加引号啊！不然可能读取不到**  
上述的顺序就是先加载```plugin1```再加载```plugin2```，最后加载```emailNotifier```。**当然，目前已经~~没有```plugin1```和`plugin2`了~~。**  

还是这个`config.yaml1`，继续添加插件需要的信息：

```yaml
emailNotifier:
  smtp: 
    host: "smtp.qq.com"                        # QQ邮箱SMTP服务器地址为例
    port: 587                                  # SMTP服务器端口
    username: "example@qq.com"                 # QQ邮箱账号
    password: "this_is_your_password"          # QQ邮箱的授权码，不支持双重验证，所以得用授权码（有些地方也叫应用密码）
    from: "example@qq.com"                     # 最好和username一样，不然有些邮箱验证有奇奇怪怪的问题
    tinmeout: 10                               # 超时时长，单位秒  
  worker:                                      # 工作协程（better版本没有这玩意）
    num: 3                                     # 工作协程数量
  consumer:                                    # 消费者（better版本也没有这玩意）
    batch_size: 10                             # 消费者批量处理数量
    idle_timeout: 1800                         # 确认超时时长，30分钟，单位秒
```

这些配置告诉插件有些东西需要啥值

### 4. 额外的插件所需

如果你的插件需要其他东西：比如从主程序传入的参数等等，请自行修改主程序传入参数；

```go
// 曾经main.go里的代码
 params := map[string]any{}

 err := extension.ExecutePlugins()
 if err != nil {
  zap.L().Error("Error executing plugins", zap.Error(err), zap.Any("params", params))
  return
 }
```

曾经的传参代码，但是似乎不太搞得好，我现在也没有精力去折腾新功能了，希望可以抛砖引玉引点好方案出来吧。  
当然也可以想配置文件中指定，就像示例里面一样，读取配置的函数。（参考```config.example.yaml```）中```emailNotifier```字段，给每一个插件建一个