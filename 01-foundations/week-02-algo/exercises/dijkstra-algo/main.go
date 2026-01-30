package main

import (
	"fmt"
	"math"
)

type Graph struct {
	vertices int
	matrix   [][]int
}

func newGraph(vertices int) *Graph {
	matrix := make([][]int, vertices)
	for i := range matrix {
		matrix[i] = make([]int, vertices)
		for j := range matrix[i] {
			matrix[i][j] = math.MaxInt32
		}
	}
	return &Graph{
		vertices: vertices,
		matrix: matrix,
	}
}

func (g *Graph) addEdge(from, to, weight int) {
	g.matrix[from][to] = weight
	g.matrix[to][from] = weight
}

func minDistance(dist []int, processed []bool) int {
	min := math.MaxInt32
	minIndex := -1

	for v := 0; v < len(dist); v++ {
		if !processed[v] && dist[v] <= min {
			min = dist[v]
			minIndex = v
		}

	}
	return minIndex
}

func dijkstra(graph *Graph, src int) []int {
	dist := make([]int, graph.vertices) 
	processed := make([]bool, graph.vertices)

	for i := range dist {
		dist[i] = math.MaxInt32
	}

	dist[src] = 0

	for count := 0; count < graph.vertices-1; count++ {
		u := minDistance(dist, processed)
		processed[u] = true

		for v := 0; v < graph.vertices; v++ {
			if !processed[v] && graph.matrix[u][v] != math.MaxInt32 && dist[u]+graph.matrix[u][v] < dist[v] {
                dist[v] = dist[u] + graph.matrix[u][v]
            }
		}
	}
	return dist
}

func main() {
    // Create a new graph with 5 vertices
    graph := newGraph(5)

    // Add weighted edges to the graph
    graph.addEdge(0, 1, 2)
    graph.addEdge(0, 2, 4)
    graph.addEdge(1, 2, 1)
    graph.addEdge(1, 3, 7)
    graph.addEdge(2, 4, 3)
    graph.addEdge(3, 4, 1)

    // Define the source vertex for Dijkstra's algorithm
    source := 0

    // Run Dijkstra's algorithm to find the shortest distances from the source vertex
    shortestDistances := dijkstra(graph, source)

    // Print the shortest distances from the source vertex to all other vertices
    fmt.Println("Shortest distances from vertex", source, "to all other vertices:")
    for v, distance := range shortestDistances {
        fmt.Printf("Vertex %d: Distance %d\n", v, distance)
    }
}