package plugintest

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
)

type PropertyCase[T any] struct {
	Seed  uint64
	Index int
	Value T
}

type PropertyGenerator struct {
	seed    uint64
	index   int
	random  *rand.Rand
	counter uint64
}

func NewPropertyGenerator(seed uint64) *PropertyGenerator {
	return &PropertyGenerator{
		seed:   seed,
		random: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
	}
}

func GeneratePropertyCases[T any](seed uint64, count int, generate func(*PropertyGenerator) T) ([]PropertyCase[T], error) {
	if count < 1 {
		return nil, errors.New("property fixture requires at least one case")
	}
	if generate == nil {
		return nil, errors.New("property fixture generator is required")
	}
	generator := NewPropertyGenerator(seed)
	cases := make([]PropertyCase[T], count)
	for index := range count {
		generator.index = index
		cases[index] = PropertyCase[T]{Seed: seed, Index: index, Value: generate(generator)}
	}
	return cases, nil
}

func CheckProperty[T any](cases []PropertyCase[T], check func(PropertyCase[T]) error) error {
	if check == nil {
		return errors.New("property fixture check is required")
	}
	for _, item := range cases {
		if err := check(item); err != nil {
			return fmt.Errorf("property failed seed=%d case=%d: %w", item.Seed, item.Index, err)
		}
	}
	return nil
}

func (g *PropertyGenerator) Seed() uint64 {
	if g == nil {
		return 0
	}
	return g.seed
}

func (g *PropertyGenerator) CaseIndex() int {
	if g == nil {
		return 0
	}
	return g.index
}

func (g *PropertyGenerator) IntN(maximum int) int {
	if g == nil || maximum <= 0 {
		return 0
	}
	return g.random.IntN(maximum)
}

func (g *PropertyGenerator) Bool() bool {
	return g != nil && g.random.IntN(2) == 1
}

func (g *PropertyGenerator) Pick(values ...string) string {
	if g == nil || len(values) == 0 {
		return ""
	}
	return values[g.random.IntN(len(values))]
}

func (g *PropertyGenerator) ID(prefix string) string {
	if g == nil {
		return strings.TrimSpace(prefix)
	}
	g.counter++
	prefix = strings.Trim(strings.ToLower(strings.TrimSpace(prefix)), "-")
	if prefix == "" {
		prefix = "fixture"
	}
	return prefix + "-" + strconv.FormatUint(g.seed, 36) + "-" + strconv.Itoa(g.index) + "-" + strconv.FormatUint(g.counter, 36)
}
