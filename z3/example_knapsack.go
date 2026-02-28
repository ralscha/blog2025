package main

import (
	"fmt"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runKnapsack() error {
	ctx := z3.NewContext()
	opt := ctx.NewOptimize()

	items := []struct {
		name   string
		weight int64
		value  int64
	}{
		{"laptop", 3, 10},
		{"camera", 2, 8},
		{"phone", 1, 5},
		{"book", 2, 3},
		{"snacks", 1, 2},
		{"headphones", 1, 4},
	}

	capacity := 6
	take := make([]*z3.Expr, len(items))
	for i := range items {
		take[i] = ctx.MkIntConst("take_" + items[i].name)
		opt.Assert(ctx.MkGe(take[i], ctx.MkInt(0, ctx.MkIntSort())))
		opt.Assert(ctx.MkLe(take[i], ctx.MkInt(1, ctx.MkIntSort())))
	}

	weightTerms := make([]*z3.Expr, len(items))
	valueTerms := make([]*z3.Expr, len(items))
	for i := range items {
		weightTerms[i] = ctx.MkMul(take[i], ctx.MkInt64(items[i].weight, ctx.MkIntSort()))
		valueTerms[i] = ctx.MkMul(take[i], ctx.MkInt64(items[i].value, ctx.MkIntSort()))
	}

	totalWeight := ctx.MkAdd(weightTerms...)
	totalValue := ctx.MkAdd(valueTerms...)

	opt.Assert(ctx.MkLe(totalWeight, ctx.MkInt(capacity, ctx.MkIntSort())))
	objective := opt.Maximize(totalValue)

	if status := opt.Check(); status != z3.Satisfiable {
		return fmt.Errorf("%s returned %s", "knapsack", status.String())
	}

	model := opt.Model()
	if upper := opt.GetUpper(objective); upper != nil {
		fmt.Printf("Maximum value: %s\n", upper.String())
	}

	usedWeight := int64(0)
	fmt.Println("Selected items:")
	for i := range items {
		takeVal, takeOk := model.Eval(take[i], true)
		if !takeOk {
			return fmt.Errorf("failed to evaluate %s", take[i].String())
		}
		takeNum, err := parseIntExpr(takeVal.String())
		if err != nil {
			return err
		}
		if takeNum == 1 {
			fmt.Printf("  %s (weight: %d, value: %d)\n", items[i].name, items[i].weight, items[i].value)
			usedWeight += items[i].weight
		}
	}
	fmt.Printf("Total weight: %d / %d\n", usedWeight, capacity)
	return nil
}
