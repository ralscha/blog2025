package main

import (
	"fmt"
	"math/rand/v2"
	"slices"
)

const (
	target         = "BRAIN"
	alphabet       = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	populationSize = 100
	mutationRate   = 0.03
	maxGenerations = 1000
	tournamentSize = 4
)

type individual struct {
	genome  string
	fitness int
}

func main() {
	population := makeInitialPopulation()
	evaluateAndSort(population)

	for generation := 0; generation <= maxGenerations; generation++ {
		best := population[0]
		fmt.Printf("Generation %3d | best %s | fitness %d/%d\n", generation, best.genome, best.fitness, 10*len(target))

		if best.genome == target {
			fmt.Println("Evolution complete.")
			return
		}

		population = nextGeneration(population)
		evaluateAndSort(population)
	}

	best := population[0]
	fmt.Printf("Stopped after %d generations. Best genome: %s (%d/%d)\n", maxGenerations, best.genome, best.fitness, 10*len(target))
}

func makeInitialPopulation() []individual {
	population := make([]individual, populationSize)
	for index := range population {
		population[index] = individual{genome: randomGenome()}
	}
	return population
}

func randomGenome() string {
	bytes := make([]byte, len(target))
	for index := range bytes {
		bytes[index] = randomLetter()
	}
	return string(bytes)
}

func nextGeneration(population []individual) []individual {
	next := make([]individual, populationSize)
	next[0] = individual{genome: population[0].genome}

	for index := 1; index < populationSize; index++ {
		parentA := tournamentSelect(population)
		parentB := tournamentSelect(population)
		child := crossover(parentA.genome, parentB.genome)
		next[index] = individual{genome: mutate(child)}
	}

	return next
}

func tournamentSelect(population []individual) individual {
	best := population[rand.IntN(len(population))]
	for draw := 1; draw < tournamentSize; draw++ {
		challenger := population[rand.IntN(len(population))]
		if challenger.fitness > best.fitness {
			best = challenger
		}
	}
	return best
}

func crossover(parentA string, parentB string) string {
	cut := rand.IntN(len(target)-1) + 1
	return parentA[:cut] + parentB[cut:]
}

func mutate(genome string) string {
	bytes := []byte(genome)
	for index := range bytes {
		if rand.Float64() < mutationRate {
			bytes[index] = randomLetter()
		}
	}
	return string(bytes)
}

func randomLetter() byte {
	return alphabet[rand.IntN(len(alphabet))]
}

func evaluateAndSort(population []individual) {
	for index := range population {
		population[index].fitness = fitness(population[index].genome)
	}
	slices.SortFunc(population, func(left, right individual) int {
		switch {
		case left.fitness > right.fitness:
			return -1
		case left.fitness < right.fitness:
			return 1
		default:
			return 0
		}
	})
}

func fitness(genome string) int {
	score := 0
	matchedGenome := make([]bool, len(target))
	matchedTarget := make([]bool, len(target))

	// First pass: correct letter at correct position -> 10 points
	for index := range target {
		if genome[index] == target[index] {
			score += 10
			matchedGenome[index] = true
			matchedTarget[index] = true
		}
	}

	// Second pass: correct letter at wrong position -> 1 point
	for genomeIdx := range genome {
		if matchedGenome[genomeIdx] {
			continue
		}
		for targetIdx := range target {
			if matchedTarget[targetIdx] {
				continue
			}
			if genome[genomeIdx] == target[targetIdx] {
				score += 1
				matchedTarget[targetIdx] = true
				break
			}
		}
	}
	return score
}
