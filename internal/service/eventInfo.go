package service

import (
	"QA-System/plugins"
	"time"
)

// FromSurveyIDToMsg 通过问卷ID将问卷信息发送到邮件提醒插件
func FromSurveyIDToMsg(surveyID int) error {
	// 获取问卷信息
	survey, err := GetSurveyByID(surveyID)
	if err != nil {
		return err
	}

	creatorEmail, err := GetUserEmailByID(survey.UserID)
	if err != nil {
		return err
	}
	// 构造消息数据
	data := map[string]any{
		"creator_email": creatorEmail,
		"survey_title":  survey.Title,
		"timestamp":     time.Now().UnixNano(),
	}

	// 使用 BetterEmailNotifier 发送邮件
	err = plugins.BetterEmailNotify(data)

	return err
}
