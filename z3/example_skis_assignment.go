package main

import (
	"fmt"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runSkisAssignment() error {
	ctx := z3.NewContext()
	opt := ctx.NewOptimize()

	skiSizes := []int64{1, 2, 5, 7, 13, 21}
	skierHeights := []int64{3, 4, 7, 11, 18}

	assignments := make([]*z3.Expr, len(skierHeights))
	for i := range skierHeights {
		assignments[i] = ctx.MkIntConst(fmt.Sprintf("ski_for_skier_%d", i))
		opt.Assert(ctx.MkGe(assignments[i], ctx.MkInt(0, ctx.MkIntSort())))
		opt.Assert(ctx.MkLt(assignments[i], ctx.MkInt(len(skiSizes), ctx.MkIntSort())))
	}
	opt.Assert(ctx.MkDistinct(assignments...))

	disparities := make([]*z3.Expr, len(skierHeights))
	for i, height := range skierHeights {
		disparities[i] = ctx.MkIntConst(fmt.Sprintf("disparity_%d", i))
		opt.Assert(ctx.MkGe(disparities[i], ctx.MkInt(0, ctx.MkIntSort())))

		for j, skiSize := range skiSizes {
			diff := skiSize - height
			if diff < 0 {
				diff = -diff
			}
			condition := ctx.MkEq(assignments[i], ctx.MkInt(j, ctx.MkIntSort()))
			opt.Assert(ctx.MkImplies(condition, ctx.MkEq(disparities[i], ctx.MkInt64(diff, ctx.MkIntSort()))))
		}
	}

	totalDisparity := ctx.MkAdd(disparities...)
	objective := opt.Minimize(totalDisparity)

	if status := opt.Check(); status != z3.Satisfiable {
		return fmt.Errorf("%s returned %s", "skis-assignment", status.String())
	}

	model := opt.Model()
	upper := opt.GetUpper(objective)
	if upper != nil {
		fmt.Printf("Minimum total disparity: %s\n", upper.String())
	}

	for i, height := range skierHeights {
		assignVal, assignOk := model.Eval(assignments[i], true)
		if !assignOk {
			return fmt.Errorf("failed to evaluate %s", assignments[i].String())
		}
		assignIndex, err := parseIntExpr(assignVal.String())
		if err != nil {
			return err
		}
		dispVal, dispOk := model.Eval(disparities[i], true)
		if !dispOk {
			return fmt.Errorf("failed to evaluate %s", disparities[i].String())
		}
		disp, err := parseIntExpr(dispVal.String())
		if err != nil {
			return err
		}
		fmt.Printf("Skier %d (height %d) gets ski of size %d (disparity: %d)\n", i, height, skiSizes[assignIndex], disp)
	}
	return nil
}
