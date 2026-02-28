package main

import (
	"fmt"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runEinsteinRiddle() error {
	ctx := z3.NewContext()
	solver := ctx.NewSolver()

	englishman := ctx.MkIntConst("Englishman")
	spaniard := ctx.MkIntConst("Spaniard")
	ukrainian := ctx.MkIntConst("Ukrainian")
	norwegian := ctx.MkIntConst("Norwegian")
	japanese := ctx.MkIntConst("Japanese")
	nationalities := []*z3.Expr{englishman, spaniard, ukrainian, norwegian, japanese}

	parliaments := ctx.MkIntConst("Parliaments")
	kools := ctx.MkIntConst("Kools")
	luckyStrike := ctx.MkIntConst("LuckyStrike")
	oldGold := ctx.MkIntConst("OldGold")
	chesterfields := ctx.MkIntConst("Chesterfields")
	cigarettes := []*z3.Expr{parliaments, kools, luckyStrike, oldGold, chesterfields}

	fox := ctx.MkIntConst("Fox")
	horse := ctx.MkIntConst("Horse")
	zebra := ctx.MkIntConst("Zebra")
	dog := ctx.MkIntConst("Dog")
	snails := ctx.MkIntConst("Snails")
	animals := []*z3.Expr{fox, horse, zebra, dog, snails}

	coffee := ctx.MkIntConst("Coffee")
	milk := ctx.MkIntConst("Milk")
	orangeJuice := ctx.MkIntConst("OrangeJuice")
	tea := ctx.MkIntConst("Tea")
	water := ctx.MkIntConst("Water")
	drinks := []*z3.Expr{coffee, milk, orangeJuice, tea, water}

	red := ctx.MkIntConst("Red")
	green := ctx.MkIntConst("Green")
	ivory := ctx.MkIntConst("Ivory")
	blue := ctx.MkIntConst("Blue")
	yellow := ctx.MkIntConst("Yellow")
	colors := []*z3.Expr{red, green, ivory, blue, yellow}

	groups := [][]*z3.Expr{nationalities, cigarettes, animals, drinks, colors}

	neighbor := func(a, b *z3.Expr) *z3.Expr {
		delta := ctx.MkSub(a, b)
		return ctx.MkOr(ctx.MkEq(delta, ctx.MkInt(1, ctx.MkIntSort())), ctx.MkEq(delta, ctx.MkInt(-1, ctx.MkIntSort())))
	}

	for _, group := range groups {
		for _, v := range group {
			solver.Assert(ctx.MkGe(v, ctx.MkInt(1, ctx.MkIntSort())))
			solver.Assert(ctx.MkLe(v, ctx.MkInt(5, ctx.MkIntSort())))
		}
		solver.Assert(ctx.MkDistinct(group...))
	}

	solver.Assert(ctx.MkEq(englishman, red))
	solver.Assert(ctx.MkEq(spaniard, dog))
	solver.Assert(ctx.MkEq(coffee, green))
	solver.Assert(ctx.MkEq(ukrainian, tea))
	solver.Assert(ctx.MkEq(green, ctx.MkAdd(ivory, ctx.MkInt(1, ctx.MkIntSort()))))
	solver.Assert(ctx.MkEq(oldGold, snails))
	solver.Assert(ctx.MkEq(kools, yellow))
	solver.Assert(ctx.MkEq(milk, ctx.MkInt(3, ctx.MkIntSort())))
	solver.Assert(ctx.MkEq(norwegian, ctx.MkInt(1, ctx.MkIntSort())))
	solver.Assert(neighbor(chesterfields, fox))
	solver.Assert(neighbor(kools, horse))
	solver.Assert(ctx.MkEq(luckyStrike, orangeJuice))
	solver.Assert(ctx.MkEq(japanese, parliaments))
	solver.Assert(neighbor(norwegian, blue))

	if status := solver.Check(); status != z3.Satisfiable {
		return fmt.Errorf("%s returned %s", "einstein", status.String())
	}

	model := solver.Model()
	waterVal, waterOk := model.Eval(water, true)
	if !waterOk {
		return fmt.Errorf("failed to evaluate %s", water.String())
	}
	waterHouse, err := parseIntExpr(waterVal.String())
	if err != nil {
		return err
	}
	zebraVal, zebraOk := model.Eval(zebra, true)
	if !zebraOk {
		return fmt.Errorf("failed to evaluate %s", zebra.String())
	}
	zebraHouse, err := parseIntExpr(zebraVal.String())
	if err != nil {
		return err
	}

	names := []string{"Englishman", "Spaniard", "Ukrainian", "Norwegian", "Japanese"}
	var waterDrinker, zebraOwner string
	for i, nat := range nationalities {
		houseVal, houseOk := model.Eval(nat, true)
		if !houseOk {
			return fmt.Errorf("failed to evaluate %s", nat.String())
		}
		house, err := parseIntExpr(houseVal.String())
		if err != nil {
			return err
		}
		if house == waterHouse {
			waterDrinker = names[i]
		}
		if house == zebraHouse {
			zebraOwner = names[i]
		}
	}

	fmt.Printf("Who drinks water? %s\n", waterDrinker)
	fmt.Printf("Who owns the zebra? %s\n", zebraOwner)
	if waterDrinker != "Norwegian" || zebraOwner != "Japanese" {
		return fmt.Errorf("unexpected result: water=%s zebra=%s", waterDrinker, zebraOwner)
	}
	return nil
}
