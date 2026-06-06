package adapter

import (
	"context"
	"fmt"
	"log"
)

// MockDevRolloutExecutor 灰度执行器的 mock 实现，打印结构化日志模拟网关调用。
type MockDevRolloutExecutor struct{}

// ApplyRollout 模拟灰度发布，打印结构化日志并返回成功。
func (m *MockDevRolloutExecutor) ApplyRollout(ctx context.Context, strategy DevRolloutStrategy) (*DevRolloutResult, error) {
	log.Printf("[mock-rollout] pluginID=%s strategyType=%s targetEnv=%s percent=%d tags=%v canaryVersion=%s taskID=%s actorID=%s",
		strategy.PluginID, strategy.StrategyType, strategy.TargetEnv,
		strategy.Percent, strategy.Tags, strategy.CanaryVersion,
		strategy.TaskID, strategy.ActorID,
	)

	previous := &DevRolloutStrategy{
		PluginID:     strategy.PluginID,
		StrategyType: strategy.StrategyType,
		TargetEnv:    strategy.TargetEnv,
	}

	return &DevRolloutResult{
		Success:       true,
		PreviousState: previous,
		Message:       fmt.Sprintf("mock rollout applied: plugin=%s strategy=%s env=%s", strategy.PluginID, strategy.StrategyType, strategy.TargetEnv),
	}, nil
}

// ApplyRollback 模拟回滚，打印结构化日志并返回成功。
func (m *MockDevRolloutExecutor) ApplyRollback(ctx context.Context, strategy DevRolloutStrategy, target DevRolloutStrategy) (*DevRolloutResult, error) {
	log.Printf("[mock-rollback] pluginID=%s strategyType=%s targetEnv=%s from=%d to=%d taskID=%s actorID=%s",
		strategy.PluginID, strategy.StrategyType, strategy.TargetEnv,
		strategy.Percent, target.Percent,
		strategy.TaskID, strategy.ActorID,
	)

	return &DevRolloutResult{
		Success:       true,
		PreviousState: &strategy,
		Message:       fmt.Sprintf("mock rollback applied: plugin=%s strategy=%s env=%s", strategy.PluginID, strategy.StrategyType, strategy.TargetEnv),
	}, nil
}

// VerifyRollout 模拟验证灰度状态，始终返回成功。
func (m *MockDevRolloutExecutor) VerifyRollout(ctx context.Context, pluginID string, targetEnv string) (*DevRolloutResult, error) {
	log.Printf("[mock-verify] pluginID=%s targetEnv=%s status=healthy", pluginID, targetEnv)

	return &DevRolloutResult{
		Success: true,
		Message: fmt.Sprintf("mock verify: plugin=%s env=%s is healthy", pluginID, targetEnv),
	}, nil
}
