package main

import (
	"fmt"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runBuildingAHouse() error {
	ctx := z3.NewContext()
	opt := ctx.NewOptimize()

	const (
		masonry = iota
		carpentry
		plumbing
		ceiling
		roofing
		painting
		windows
		facade
		garden
		moving
		numTasks
	)

	taskNames := []string{"masonry", "carpentry", "plumbing", "ceiling", "roofing", "painting", "windows", "facade", "garden", "moving"}
	durations := []int64{35, 15, 40, 15, 5, 10, 5, 10, 5, 5}

	var totalDuration int64
	for _, d := range durations {
		totalDuration += d
	}

	start := make([]*z3.Expr, numTasks)
	end := make([]*z3.Expr, numTasks)
	for i := range numTasks {
		start[i] = ctx.MkIntConst("start_" + taskNames[i])
		end[i] = ctx.MkIntConst("end_" + taskNames[i])

		opt.Assert(ctx.MkGe(start[i], ctx.MkInt(0, ctx.MkIntSort())))
		opt.Assert(ctx.MkLe(start[i], ctx.MkInt64(totalDuration, ctx.MkIntSort())))
		opt.Assert(ctx.MkGe(end[i], ctx.MkInt(0, ctx.MkIntSort())))
		opt.Assert(ctx.MkLe(end[i], ctx.MkInt64(totalDuration, ctx.MkIntSort())))
		opt.Assert(ctx.MkEq(end[i], ctx.MkAdd(start[i], ctx.MkInt64(durations[i], ctx.MkIntSort()))))
	}

	makespan := ctx.MkIntConst("makespan")
	opt.Assert(ctx.MkGe(makespan, ctx.MkInt(0, ctx.MkIntSort())))
	opt.Assert(ctx.MkLe(makespan, ctx.MkInt64(totalDuration, ctx.MkIntSort())))
	for i := range numTasks {
		opt.Assert(ctx.MkGe(makespan, end[i]))
	}

	precedences := [][2]int{
		{masonry, carpentry},
		{masonry, plumbing},
		{masonry, ceiling},
		{carpentry, roofing},
		{ceiling, painting},
		{roofing, windows},
		{roofing, facade},
		{plumbing, facade},
		{roofing, garden},
		{plumbing, garden},
		{windows, moving},
		{facade, moving},
		{garden, moving},
		{painting, moving},
	}
	for _, p := range precedences {
		x, y := p[0], p[1]
		opt.Assert(ctx.MkLe(ctx.MkAdd(start[x], ctx.MkInt64(durations[x], ctx.MkIntSort())), start[y]))
	}

	objective := opt.Minimize(makespan)
	if status := opt.Check(); status != z3.Satisfiable {
		return fmt.Errorf("%s returned %s", "building-a-house", status.String())
	}

	model := opt.Model()
	if lower := opt.GetLower(objective); lower != nil {
		fmt.Printf("Minimum makespan: %s\n", lower.String())
	}

	fmt.Println("Schedule:")
	for i := range numTasks {
		sVal, sOk := model.Eval(start[i], true)
		if !sOk {
			return fmt.Errorf("failed to evaluate %s", start[i].String())
		}
		s, err := parseIntExpr(sVal.String())
		if err != nil {
			return err
		}
		eVal, eOk := model.Eval(end[i], true)
		if !eOk {
			return fmt.Errorf("failed to evaluate %s", end[i].String())
		}
		e, err := parseIntExpr(eVal.String())
		if err != nil {
			return err
		}
		fmt.Printf("  %-10s: %3d -- (%2d) --> %3d\n", taskNames[i], s, durations[i], e)
	}

	makespanVal, makespanOk := model.Eval(makespan, true)
	if !makespanOk {
		return fmt.Errorf("failed to evaluate %s", makespan.String())
	}
	makespanNum, err := parseIntExpr(makespanVal.String())
	if err != nil {
		return err
	}
	if makespanNum != 90 {
		return fmt.Errorf("expected makespan 90, got %d", makespanNum)
	}
	return nil
}
