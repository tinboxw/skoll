package adapter

import "context"

// DevRolloutStrategy 灰度策略参数
type DevRolloutStrategy struct {
	PluginID      string
	StrategyType  string // percent / tag / canary
	TargetEnv     string // production / staging
	Percent       int
	Tags          []string
	CanaryVersion string
	TaskID        string
	ActorID       string
}

// DevRolloutResult 灰度执行结果
type DevRolloutResult struct {
	Success       bool
	PreviousState *DevRolloutStrategy // 回滚时记录之前的状态
	Message       string
}

// DevRolloutExecutor 灰度执行器接口
type DevRolloutExecutor interface {
	ApplyRollout(ctx context.Context, strategy DevRolloutStrategy) (*DevRolloutResult, error)
	ApplyRollback(ctx context.Context, strategy DevRolloutStrategy, target DevRolloutStrategy) (*DevRolloutResult, error)
	VerifyRollout(ctx context.Context, pluginID string, targetEnv string) (*DevRolloutResult, error)
}
