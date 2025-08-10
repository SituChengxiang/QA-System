package plugins

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"QA-System/internal/global/config"
	"QA-System/pkg/extension"

	"github.com/bytedance/gopkg/util/gopool"
	"gopkg.in/gomail.v2"
)

// emailNotifier 插件需要的基本信息
type emailNotifier struct {
	smtpHost     string         // SMTP服务器地址
	smtpPort     int            // SMTP服务器端口
	smtpUsername string         // SMTP服务器用户名
	smtpPassword string         // SMTP服务器密码
	from         string         // 发件人地址
	workerNum    int32          // 工作协程数量
	pool         gopool.Pool    // 协程池
	dialer       *gomail.Dialer // 邮件发送器
	logger       *slog.Logger   // 日志记录器
	enabled      bool           // 插件是否启用
}

var betterNotifier *emailNotifier

// init 注册插件
func init() {
	betterNotifier = &emailNotifier{
		workerNum: 20, // 默认20个协程
		logger:    extension.GetPluginLogger(),
		enabled:   false, // 默认禁用，需要配置验证后启用
	}

	if err := betterNotifier.initialize(); err != nil {
		betterNotifier.logger.Warn("Failed to initialize email_notifier", "error", err)
		// 即使初始化失败也注册插件，但标记为禁用状态
	}

	if err := extension.RegisterPlugin(betterNotifier); err != nil {
		betterNotifier.logger.Warn("Failed to register email_notifier", "error", err)
		return
	}
}

// initialize 从配置文件中读取配置信息
func (p *emailNotifier) initialize() error {
	// 读取SMTP配置
	p.smtpHost = config.Config.GetString("emailNotifier.smtp.host")
	p.smtpPort = config.Config.GetInt("emailNotifier.smtp.port")
	p.smtpUsername = config.Config.GetString("emailNotifier.smtp.username")
	p.smtpPassword = config.Config.GetString("emailNotifier.smtp.password")
	p.from = config.Config.GetString("emailNotifier.smtp.from")
	if config.Config.IsSet("emailNotifier.workerNum") {
		if workers := config.Config.GetInt32("emailNotifier.workerNum"); workers > 0 && workers < 2147483647 {
			p.workerNum = workers
		}
	}

	// 检查关键配置是否存在
	if p.smtpHost == "" || p.smtpUsername == "" || p.smtpPassword == "" || p.from == "" {
		p.enabled = false
		return errors.New("incomplete SMTP configuration, plugin will be disabled")
	}

	// 如果端口未设置，使用默认值
	if p.smtpPort == 0 {
		p.smtpPort = 587
	}

	poolConfig := gopool.NewConfig()
	// 从配置文件中读取 ScaleThreshold，如果未设置或小于0，或者大于int32最大值则使用默认值100
	scaleThreshold := int32(100) // 默认值
	if config.Config.IsSet("emailNotifier.scaleThreshold") {
		if shold := config.Config.GetInt32("emailNotifier.scaleThreshold"); shold > 0 && shold < 2147483647 {
			scaleThreshold = shold
		}
	}

	poolConfig.ScaleThreshold = scaleThreshold
	p.pool = gopool.NewPool("emailPool", p.workerNum, poolConfig)

	// pool的panic处理
	p.pool.SetPanicHandler(func(ctx context.Context, panicReason any) {
		p.logger.Error("goroutine panic recovered",
			"panic_reason", panicReason,
			"context", fmt.Sprintf("%v", ctx))
	})

	// 初始化 Dialer（复用连接）
	p.dialer = gomail.NewDialer(p.smtpHost, p.smtpPort, p.smtpUsername, p.smtpPassword)
	p.enabled = true

	p.logger.Info("Email notifier initialized successfully",
		"host", p.smtpHost,
		"port", p.smtpPort,
		"from", p.from)

	return nil
}

// GetMetadata 返回插件的元数据
func (p *emailNotifier) GetMetadata() extension.PluginMetadata {
	_ = p
	return extension.PluginMetadata{
		Name:        "emailNotifier",
		Version:     "0.1.0",
		Author:      "SituChengxiang, Copilot, Qwen2.5, DeepSeek",
		Description: "Send email notifications for new survey responses",
	}
}

// Execute 启动插件，这里只是记录一下启动信息
func (p *emailNotifier) Execute(params map[string]any) error {
	// 检查插件是否已启用
	if !p.enabled {
		p.logger.Warn("Email notifier is disabled due to missing configuration")
		return errors.New("email notifier is disabled due to incomplete SMTP configuration")
	}

	// 如果没有参数，仅记录启动信息
	if params == nil {
		p.logger.Info("emailNotifier started", "workers", int(p.workerNum))
		return nil
	}

	// 如果有参数，记录参数信息
	err := p.handleParamters(params)
	return err
}

// handleParamters 发送邮件通知的核心部分
func (p *emailNotifier) handleParamters(info any) error {
	// 输入校验
	data, ok := info.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid info type: %T", info)
	}

	// 提取必要字段，接收人和问卷标题
	recipient, ok := data["creator_email"].(string)
	if !ok || recipient == "" {
		return fmt.Errorf("invalid recipient: %v", data["creator_email"])
	}

	title, ok := data["survey_title"].(string)
	if !ok || title == "" {
		return fmt.Errorf("invalid title: %v", data["survey_title"])
	}

	p.pool.CtxGo(context.Background(), func() {
		defer func() {
			if r := recover(); r != nil {
				p.logger.Error("邮件任务 Panic", "panic_reason", r)
			}
		}()

		if err := p.SendEmail(map[string]any{
			"recipient": recipient,
			"title":     title,
		}); err != nil {
			p.logger.Error("email sent failed",
				"recipient", recipient,
				"title", title,
				"error", err)
		}
	})
	return nil
}

// SendEmail 发送邮件
func (p *emailNotifier) SendEmail(data map[string]any) error {
	// 参数校验
	recipient, ok := data["recipient"].(string)
	if !ok || recipient == "" {
		p.logger.Info("Recipient email is empty, skip current sending email",
			"data", fmt.Sprintf("%v", data)) // 记录完整数据便于排查
		return nil
	}

	title, ok := data["title"].(string)
	if !ok {
		return errors.New("invalid title type")
	}

	// 创建邮件
	m := gomail.NewMessage()
	m.SetHeader("From", p.from)
	m.SetAddressHeader("To", recipient, "尊敬的用户")
	// m.SetAddressHeader("Cc", p.from, "QA-System") 如果不放心可以换成这个，会在发件邮箱那里增量存一份
	m.SetHeader("Subject", fmt.Sprintf("您的问卷\"%s\"收到了新回复", title))
	m.SetBody("text/plain", fmt.Sprintf("您的问卷\"%s\"收到了新回复，请及时查收。", title))

	// 复用 Dialer发送邮件
	if err := p.dialer.DialAndSend(m); err != nil {
		return err
	}

	p.logger.Info("email sent successfully",
		"recipient", recipient,
		"title", title)
	return nil
}

// IsHealthy 实现插件健康检查接口
func (p *emailNotifier) IsHealthy() bool {
	return p.enabled && p.dialer != nil
}

// GetStatus 获取插件状态描述
func (p *emailNotifier) GetStatus() string {
	if !p.enabled {
		return "disabled: incomplete SMTP configuration"
	}
	if p.dialer == nil {
		return "error: SMTP dialer not initialized"
	}
	return "active: ready to send emails"
}
