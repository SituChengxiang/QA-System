package plugins

import (
	"context"
	"errors"
	"fmt"

	"QA-System/internal/global/config"
	"QA-System/pkg/extension"
	"github.com/bytedance/gopkg/util/gopool"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

// BetterEmailNotifier 插件需要的基本信息
type BetterEmailNotifier struct {
	smtpHost     string         // SMTP服务器地址
	smtpPort     int            // SMTP服务器端口
	smtpUsername string         // SMTP服务器用户名
	smtpPassword string         // SMTP服务器密码
	from         string         // 发件人地址
	workerNum    int32          // 工作协程数量
	pool         gopool.Pool    // 协程池
	dialer       *gomail.Dialer // 邮件发送器
}

var betterNotifier *BetterEmailNotifier

// init 注册插件
func init() {
	betterNotifier = &BetterEmailNotifier{workerNum: 20} // 默认20个协程
	if err := betterNotifier.initialize(); err != nil {
		zap.L().Warn("Failed to initialize email_notifier,", zap.Any("error:", err))
		return
	}
	extension.GetDefaultManager().RegisterPlugin(betterNotifier)
}

// initialize 从配置文件中读取配置信息
func (p *BetterEmailNotifier) initialize() error {
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

	if p.smtpHost == "" || p.smtpUsername == "" || p.smtpPassword == "" || p.from == "" {
		return errors.New("invalid SMTP configuration, the plugin won't work")
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
		zap.L().Error("goroutine panic recovered",
			zap.Any("panic_reason", panicReason),
			zap.Any("context", ctx),
			zap.Stack("stack_trace"))
	})

	// 初始化 Dialer（复用连接）
	p.dialer = gomail.NewDialer(p.smtpHost, p.smtpPort, p.smtpUsername, p.smtpPassword)

	return nil
}

// GetMetadata 返回插件的元数据
func (p *BetterEmailNotifier) GetMetadata() extension.PluginMetadata {
	_ = p
	return extension.PluginMetadata{
		Name:        "betterEmailNotifier",
		Version:     "0.1.0",
		Author:      "SituChengxiang, Copilot, Qwen2.5, DeepSeek",
		Description: "Send email notifications for new survey responses",
	}
}

// Execute 启动插件，这里只是记录一下启动信息
func (p *BetterEmailNotifier) Execute() error {
	zap.L().Info("BetterEmailNotifier started", zap.Int("workers", int(p.workerNum)))
	return nil
}

// BetterEmailNotify 发送邮件通知的核心部分
func BetterEmailNotify(info any) error {
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

	betterNotifier.pool.CtxGo(context.Background(), func() {
		defer func() {
			if r := recover(); r != nil {
				zap.L().Error("邮件任务 Panic",
					zap.Any("panic_reason", r),
					zap.Stack("stack_trace"))
			}
		}()

		if err := betterNotifier.SendEmail(map[string]any{
			"recipient": recipient,
			"title":     title,
		}); err != nil {
			zap.L().Error("email sent failed",
				zap.String("recipient", recipient),
				zap.String("title", title),
				zap.Any("error:", err))
		}
	})
	return nil
}

// SendEmail 发送邮件
func (p *BetterEmailNotifier) SendEmail(data map[string]any) error {
	// 参数校验
	recipient, ok := data["recipient"].(string)
	if !ok || recipient == "" {
		zap.L().Info("Recipient email is empty, skip current sending email",
			zap.Any("data", data)) // 记录完整数据便于排查
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

	zap.L().Info("email sent successfully",
		zap.String("recipient", recipient),
		zap.String("title", title))
	return nil
}
