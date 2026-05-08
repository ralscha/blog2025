package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
)

const (
	populationSize     = 250
	tournamentSize     = 5
	eliteCount         = 8
	mutationRate       = 0.22
	progressInterval   = 5
	convergenceWindow  = 50
	convergenceEpsilon = 0.001
)

type city struct {
	name string
	x    float64
	y    float64
}

type candidate struct {
	route    []int
	distance float64
	fitness  float64
}

var cities = []city{
	{name: "London", x: 2, y: 7},
	{name: "Paris", x: 3.5, y: 5.5},
	{name: "Amsterdam", x: 4.5, y: 7},
	{name: "Berlin", x: 7, y: 6.5},
	{name: "Munich", x: 6.5, y: 5},
	{name: "Vienna", x: 8.5, y: 5},
	{name: "Rome", x: 6.5, y: 3},
	{name: "Madrid", x: 1.5, y: 3.5},
	{name: "Barcelona", x: 3, y: 3},
	{name: "Warsaw", x: 9.5, y: 6},
	{name: "Stockholm", x: 8, y: 9},
	{name: "Prague", x: 7.5, y: 5.5},
}

func main() {
	population := makeInitialPopulation()
	evaluatePopulation(population)
	sortPopulation(population)

	startBest := population[0]
	fmt.Printf("Initial best distance: %.2f\n", startBest.distance)
	fmt.Printf("Initial route: %s\n\n", formatRoute(startBest.route))

	bestDistance := startBest.distance
	stagnantGenerations := 0
	finalGeneration := 0

	for generation := 1; ; generation++ {
		population = nextGeneration(population)
		evaluatePopulation(population)
		sortPopulation(population)
		finalGeneration = generation

		best := population[0]
		if best.distance < bestDistance-convergenceEpsilon {
			bestDistance = best.distance
			stagnantGenerations = 0
		} else {
			stagnantGenerations++
		}

		if generation%progressInterval == 0 || generation == 1 {
			averageDistance := averageDistance(population)
			fmt.Printf(
				"Generation %3d | best distance %.2f | avg distance %.2f | stagnant %3d | route %s\n",
				generation,
				best.distance,
				averageDistance,
				stagnantGenerations,
				formatRoute(best.route),
			)
		}

		if stagnantGenerations >= convergenceWindow {
			fmt.Printf(
				"Converged after %d generations with no meaningful improvement for %d generations.\n",
				generation,
				stagnantGenerations,
			)
			break
		}
	}

	best := population[0]
	fmt.Println()
	fmt.Printf("Total generations: %d\n", finalGeneration)
	fmt.Printf("Final best distance: %.2f\n", best.distance)
	fmt.Printf("Improvement: %.2f%%\n", percentImprovement(startBest.distance, best.distance))
	fmt.Printf("Final route: %s\n", formatRoute(best.route))
	if !isValidRoute(best.route) {
		panic("best route is invalid")
	}
}

func makeInitialPopulation() []candidate {
	population := make([]candidate, populationSize)
	baseRoute := make([]int, len(cities))
	for index := range baseRoute {
		baseRoute[index] = index
	}

	for index := range population {
		route := slices.Clone(baseRoute)
		rand.Shuffle(len(route), func(left, right int) {
			route[left], route[right] = route[right], route[left]
		})
		population[index] = candidate{route: route}
	}

	return population
}

func nextGeneration(population []candidate) []candidate {
	next := make([]candidate, 0, len(population))

	for _, elite := range population[:eliteCount] {
		next = append(next, candidate{route: slices.Clone(elite.route)})
	}

	for len(next) < len(population) {
		parentA := tournamentSelect(population)
		parentB := tournamentSelect(population)

		childRoute := orderCrossover(parentA.route, parentB.route)
		mutate(childRoute)
		next = append(next, candidate{route: childRoute})
	}

	return next
}

func tournamentSelect(population []candidate) candidate {
	best := population[rand.IntN(len(population))]
	for draw := 1; draw < tournamentSize; draw++ {
		challenger := population[rand.IntN(len(population))]
		if challenger.fitness > best.fitness {
			best = challenger
		}
	}
	return best
}

func orderCrossover(parentA []int, parentB []int) []int {
	child := make([]int, len(parentA))
	for index := range child {
		child[index] = -1
	}

	left := rand.IntN(len(parentA))
	right := rand.IntN(len(parentA))
	if left > right {
		left, right = right, left
	}

	used := make([]bool, len(parentA))
	for index := left; index <= right; index++ {
		gene := parentA[index]
		child[index] = gene
		used[gene] = true
	}

	insertAt := (right + 1) % len(child)
	for offset := range parentB {
		gene := parentB[(right+1+offset)%len(parentB)]
		if used[gene] {
			continue
		}
		child[insertAt] = gene
		used[gene] = true
		insertAt = (insertAt + 1) % len(child)
	}

	return child
}

func mutate(route []int) {
	if rand.Float64() >= mutationRate {
		return
	}

	left := rand.IntN(len(route))
	right := rand.IntN(len(route))
	for left == right {
		right = rand.IntN(len(route))
	}
	route[left], route[right] = route[right], route[left]
}

func evaluatePopulation(population []candidate) {
	for index := range population {
		distance := routeDistance(population[index].route)
		population[index].distance = distance
		population[index].fitness = 1 / distance
	}
}

func sortPopulation(population []candidate) {
	slices.SortFunc(population, func(left, right candidate) int {
		switch {
		case left.distance < right.distance:
			return -1
		case left.distance > right.distance:
			return 1
		default:
			return 0
		}
	})
}

func routeDistance(route []int) float64 {
	total := 0.0
	for index := range route {
		current := cities[route[index]]
		next := cities[route[(index+1)%len(route)]]
		total += distance(current, next)
	}
	return total
}

func distance(left city, right city) float64 {
	deltaX := left.x - right.x
	deltaY := left.y - right.y
	return math.Hypot(deltaX, deltaY)
}

func averageDistance(population []candidate) float64 {
	total := 0.0
	for _, individual := range population {
		total += individual.distance
	}
	return total / float64(len(population))
}

func percentImprovement(start float64, end float64) float64 {
	return ((start - end) / start) * 100
}

func formatRoute(route []int) string {
	labels := make([]string, 0, len(route)+1)
	for _, cityIndex := range route {
		labels = append(labels, cities[cityIndex].name)
	}
	labels = append(labels, cities[route[0]].name)
	return strings.Join(labels, " -> ")
}

func isValidRoute(route []int) bool {
	if len(route) != len(cities) {
		return false
	}

	seen := make([]bool, len(cities))
	for _, cityIndex := range route {
		if cityIndex < 0 || cityIndex >= len(cities) || seen[cityIndex] {
			return false
		}
		seen[cityIndex] = true
	}
	return true
}
