package main

import (
	"fmt"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runSendMoreMoney() error {
	ctx := z3.NewContext()
	solver := ctx.NewSolver()

	s := ctx.MkIntConst("S")
	e := ctx.MkIntConst("E")
	n := ctx.MkIntConst("N")
	d := ctx.MkIntConst("D")
	m := ctx.MkIntConst("M")
	o := ctx.MkIntConst("O")
	r := ctx.MkIntConst("R")
	y := ctx.MkIntConst("Y")
	letters := []*z3.Expr{s, e, n, d, m, o, r, y}

	for _, letter := range letters {
		solver.Assert(ctx.MkGe(letter, ctx.MkInt(0, ctx.MkIntSort())))
		solver.Assert(ctx.MkLe(letter, ctx.MkInt(9, ctx.MkIntSort())))
	}
	solver.Assert(ctx.MkDistinct(letters...))
	solver.Assert(ctx.MkGe(s, ctx.MkInt(1, ctx.MkIntSort())))
	solver.Assert(ctx.MkGe(m, ctx.MkInt(1, ctx.MkIntSort())))

	send := ctx.MkAdd(ctx.MkMul(ctx.MkInt(1000, ctx.MkIntSort()), s), ctx.MkMul(ctx.MkInt(100, ctx.MkIntSort()), e), ctx.MkMul(ctx.MkInt(10, ctx.MkIntSort()), n), d)
	more := ctx.MkAdd(ctx.MkMul(ctx.MkInt(1000, ctx.MkIntSort()), m), ctx.MkMul(ctx.MkInt(100, ctx.MkIntSort()), o), ctx.MkMul(ctx.MkInt(10, ctx.MkIntSort()), r), e)
	money := ctx.MkAdd(ctx.MkMul(ctx.MkInt(10000, ctx.MkIntSort()), m), ctx.MkMul(ctx.MkInt(1000, ctx.MkIntSort()), o), ctx.MkMul(ctx.MkInt(100, ctx.MkIntSort()), n), ctx.MkMul(ctx.MkInt(10, ctx.MkIntSort()), e), y)
	solver.Assert(ctx.MkEq(ctx.MkAdd(send, more), money))

	if status := solver.Check(); status != z3.Satisfiable {
		return fmt.Errorf("%s returned %s", "send-more-money", status.String())
	}

	model := solver.Model()
	svVal, svOk := model.Eval(s, true)
	if !svOk {
		return fmt.Errorf("failed to evaluate %s", s.String())
	}
	sv, err := parseIntExpr(svVal.String())
	if err != nil {
		return err
	}
	evVal, evOk := model.Eval(e, true)
	if !evOk {
		return fmt.Errorf("failed to evaluate %s", e.String())
	}
	ev, err := parseIntExpr(evVal.String())
	if err != nil {
		return err
	}
	nvVal, nvOk := model.Eval(n, true)
	if !nvOk {
		return fmt.Errorf("failed to evaluate %s", n.String())
	}
	nv, err := parseIntExpr(nvVal.String())
	if err != nil {
		return err
	}
	dvVal, dvOk := model.Eval(d, true)
	if !dvOk {
		return fmt.Errorf("failed to evaluate %s", d.String())
	}
	dv, err := parseIntExpr(dvVal.String())
	if err != nil {
		return err
	}
	mvVal, mvOk := model.Eval(m, true)
	if !mvOk {
		return fmt.Errorf("failed to evaluate %s", m.String())
	}
	mv, err := parseIntExpr(mvVal.String())
	if err != nil {
		return err
	}
	ovVal, ovOk := model.Eval(o, true)
	if !ovOk {
		return fmt.Errorf("failed to evaluate %s", o.String())
	}
	ov, err := parseIntExpr(ovVal.String())
	if err != nil {
		return err
	}
	rvVal, rvOk := model.Eval(r, true)
	if !rvOk {
		return fmt.Errorf("failed to evaluate %s", r.String())
	}
	rv, err := parseIntExpr(rvVal.String())
	if err != nil {
		return err
	}
	yvVal, yvOk := model.Eval(y, true)
	if !yvOk {
		return fmt.Errorf("failed to evaluate %s", y.String())
	}
	yv, err := parseIntExpr(yvVal.String())
	if err != nil {
		return err
	}

	sendNum := sv*1000 + ev*100 + nv*10 + dv
	moreNum := mv*1000 + ov*100 + rv*10 + ev
	moneyNum := mv*10000 + ov*1000 + nv*100 + ev*10 + yv

	fmt.Printf("S=%d E=%d N=%d D=%d M=%d O=%d R=%d Y=%d\n", sv, ev, nv, dv, mv, ov, rv, yv)
	fmt.Printf("SEND=%d + MORE=%d = MONEY=%d\n", sendNum, moreNum, moneyNum)
	if sendNum+moreNum != moneyNum {
		return fmt.Errorf("invalid solution: %d + %d != %d", sendNum, moreNum, moneyNum)
	}
	return nil
}
