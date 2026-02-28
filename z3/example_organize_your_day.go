package main

import (
	"fmt"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runOrganizeYourDay() error {
	ctx := z3.NewContext()
	solver := ctx.NewSolver()

	workStart := ctx.MkIntConst("work_start")
	mailStart := ctx.MkIntConst("mail_start")
	bankStart := ctx.MkIntConst("bank_start")
	shoppingStart := ctx.MkIntConst("shopping_start")

	tasks := []*z3.Expr{workStart, mailStart, bankStart, shoppingStart}
	durations := []int64{4, 1, 2, 1}
	names := []string{"work", "mail", "bank", "shopping"}

	for i := range tasks {
		solver.Assert(ctx.MkGe(tasks[i], ctx.MkInt(9, ctx.MkIntSort())))
		solver.Assert(ctx.MkLe(ctx.MkAdd(tasks[i], ctx.MkInt64(durations[i], ctx.MkIntSort())), ctx.MkInt(17, ctx.MkIntSort())))
	}

	for i := range tasks {
		for j := i + 1; j < len(tasks); j++ {
			left := ctx.MkLe(ctx.MkAdd(tasks[i], ctx.MkInt64(durations[i], ctx.MkIntSort())), tasks[j])
			right := ctx.MkLe(ctx.MkAdd(tasks[j], ctx.MkInt64(durations[j], ctx.MkIntSort())), tasks[i])
			solver.Assert(ctx.MkOr(left, right))
		}
	}

	solver.Assert(ctx.MkGe(workStart, ctx.MkInt(11, ctx.MkIntSort())))
	solver.Assert(ctx.MkLe(ctx.MkAdd(mailStart, ctx.MkInt(1, ctx.MkIntSort())), workStart))
	solver.Assert(ctx.MkLe(ctx.MkAdd(bankStart, ctx.MkInt(2, ctx.MkIntSort())), shoppingStart))

	if status := solver.Check(); status != z3.Satisfiable {
		return fmt.Errorf("%s returned %s", "organize-your-day", status.String())
	}

	model := solver.Model()
	fmt.Println("Schedule:")
	for i := range tasks {
		startVal, startOk := model.Eval(tasks[i], true)
		if !startOk {
			return fmt.Errorf("failed to evaluate %s", tasks[i].String())
		}
		start, err := parseIntExpr(startVal.String())
		if err != nil {
			return err
		}
		fmt.Printf("  %s: %d:00 - %d:00 (%d hour(s))\n", names[i], start, start+durations[i], durations[i])
	}
	return nil
}
