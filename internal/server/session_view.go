package server

import "feishu-personality-agent/internal/session"

type progressView struct {
	Stage   string `json:"stage"`
	Label   string `json:"label"`
	Percent int    `json:"percent"`
}

func progressForStatus(status session.Status) progressView {
	switch status {
	case session.StatusConfigPending:
		return progressView{Stage: string(status), Label: "等待完成飞书应用配置", Percent: 20}
	case session.StatusLoginPending:
		return progressView{Stage: string(status), Label: "等待完成飞书授权", Percent: 32}
	case session.StatusAuthenticated:
		return progressView{Stage: string(status), Label: "授权完成，准备开始分析", Percent: 45}
	case session.StatusCollecting:
		return progressView{Stage: string(status), Label: "正在采集授权范围内的飞书数据", Percent: 66}
	case session.StatusAnalyzing:
		return progressView{Stage: string(status), Label: "正在生成结构化 BSPI 报告", Percent: 86}
	case session.StatusDone:
		return progressView{Stage: string(status), Label: "报告已生成", Percent: 100}
	case session.StatusFailed:
		return progressView{Stage: string(status), Label: "流程执行失败", Percent: 100}
	default:
		return progressView{Stage: string(session.StatusCreated), Label: "等待连接飞书", Percent: 5}
	}
}

func nextActionForStatus(status session.Status) string {
	switch status {
	case session.StatusConfigPending, session.StatusLoginPending:
		return "complete_authorization"
	case session.StatusAuthenticated:
		return "start_analysis"
	case session.StatusCollecting, session.StatusAnalyzing:
		return "wait"
	case session.StatusDone:
		return "view_report"
	case session.StatusFailed:
		return "retry"
	default:
		return "connect_feishu"
	}
}

func recordStatus(item *session.Session, status session.Status, label string) {
	item.Status = status
	progress := progressForStatus(status)
	item.RecordEvent(status, label, progress.Percent)
}
