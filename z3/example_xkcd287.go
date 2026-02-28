package main

import (
	"fmt"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runXKCD287() error {
	ctx := z3.NewContext()
	solver := ctx.NewSolver()

	prices := []int64{215, 275, 335, 355, 420, 580}
	itemNames := []string{"mixed_fruits", "french_fries", "side_salad", "hot_wings", "mozzarella_sticks", "sampler_plate"}
	total := int64(1505)

	quantities := make([]*z3.Expr, len(itemNames))
	for i, name := range itemNames {
		quantities[i] = ctx.MkIntConst(name)
		solver.Assert(ctx.MkGe(quantities[i], ctx.MkInt(0, ctx.MkIntSort())))
		solver.Assert(ctx.MkLe(quantities[i], ctx.MkInt(10, ctx.MkIntSort())))
	}

	sumTerms := make([]*z3.Expr, len(prices))
	for i, price := range prices {
		sumTerms[i] = ctx.MkMul(quantities[i], ctx.MkInt64(price, ctx.MkIntSort()))
	}
	solver.Assert(ctx.MkEq(ctx.MkAdd(sumTerms...), ctx.MkInt64(total, ctx.MkIntSort())))

	solutions := 0
	for {
		status := solver.Check()
		if status == z3.Unsatisfiable {
			break
		}
		if status != z3.Satisfiable {
			return fmt.Errorf("%s returned %s", "xkcd287", status.String())
		}

		solutions++
		model := solver.Model()
		fmt.Printf("Solution %d:\n", solutions)

		blocking := make([]*z3.Expr, 0, len(itemNames))
		for i, name := range itemNames {
			qtyVal, qtyOk := model.Eval(quantities[i], true)
			if !qtyOk {
				return fmt.Errorf("failed to evaluate %s", quantities[i].String())
			}
			qty, err := parseIntExpr(qtyVal.String())
			if err != nil {
				return err
			}
			if qty > 0 {
				fmt.Printf("  %d x %s = $%.2f\n", qty, name, float64(qty)*float64(prices[i])/100.0)
			}
			blocking = append(blocking, ctx.MkNot(ctx.MkEq(quantities[i], ctx.MkInt64(qty, ctx.MkIntSort()))))
		}
		solver.Assert(ctx.MkOr(blocking...))

		if solutions >= 10 {
			fmt.Println("Stopping after 10 solutions")
			break
		}
	}

	if solutions == 0 {
		return fmt.Errorf("expected at least one xkcd solution")
	}
	fmt.Printf("Found %d solution(s)\n", solutions)
	return nil
}
