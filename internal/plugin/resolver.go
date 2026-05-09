package plugin

import "fmt"

type DependencyResolver interface {
	ResolveEnableOrder(pluginID string, infos map[string]Info) ([]string, error)
	CanDisable(pluginID string, infos map[string]Info) error
}

type TopologicalResolver struct{}

func NewTopologicalResolver() *TopologicalResolver {
	return &TopologicalResolver{}
}

func (r *TopologicalResolver) ResolveEnableOrder(pluginID string, infos map[string]Info) ([]string, error) {
	if _, ok := infos[pluginID]; !ok {
		return nil, ErrPluginNotFound
	}

	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	order := make([]string, 0, len(infos))

	var dfs func(string) error
	dfs = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("%w: cycle detected at %s", ErrPluginDependency, id)
		}
		if visited[id] {
			return nil
		}

		info, ok := infos[id]
		if !ok {
			return fmt.Errorf("%w: missing dependency %s", ErrPluginDependency, id)
		}

		visiting[id] = true
		for _, dep := range info.Dependencies {
			if _, ok := infos[dep.ID]; !ok {
				return fmt.Errorf("%w: dependency %s not installed", ErrPluginDependency, dep.ID)
			}
			if err := dfs(dep.ID); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		order = append(order, id)
		return nil
	}

	if err := dfs(pluginID); err != nil {
		return nil, err
	}

	return order, nil
}

func (r *TopologicalResolver) CanDisable(pluginID string, infos map[string]Info) error {
	for _, info := range infos {
		if info.ID == pluginID || info.State != StateEnabled {
			continue
		}

		for _, dep := range info.Dependencies {
			if dep.ID == pluginID {
				return fmt.Errorf("%w: enabled plugin %s depends on %s", ErrPluginDependency, info.ID, pluginID)
			}
		}
	}

	return nil
}
