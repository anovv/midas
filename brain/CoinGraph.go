package brain

import (
	"midas/apis/binance"
	"log"
	"midas/common"
	"strconv"
	"strings"
	"math"
	"time"
	"fmt"
)

type Vertex struct {
	Coin common.Coin
}

type Edge struct {
	From *Vertex
	To *Vertex
	Weight *Weight
}

type Weight struct {
	Value float64
	DepthRecords *common.DepthRecords
}

func (w Weight) String() string {
	return common.FloatToString(w.Value)
}

var graph map[Vertex][]Edge
var distances map[Vertex]float64
var parents map[Vertex]*Vertex // TODO is this needed

func InitCoinGraph() {
	pairs, err := binance.GetAllPairs()
	if err != nil {
		return
	}

	graph = make(map[Vertex][]Edge)

	for _, pair := range pairs {
		// TODO make everything use pointers
		VertexA := Vertex{pair.CoinA}
		VertexB := Vertex{pair.CoinB}
		_, hasKeyA := graph[VertexA]
		_, hasKeyB := graph[VertexB]
		var weightTo Weight
		var weightFrom Weight
		if strings.Compare(VertexA.Coin.String(), "GVT") == 0 &&
			strings.Compare(VertexB.Coin.String(), "BTC") == 0 {
			weightTo = Weight{1.2323, nil}
			weightFrom = Weight{-2.2323333, nil}
		//} else if strings.Compare(VertexA.Coin.String(), "EOS") == 0 &&
		//	strings.Compare(VertexB.Coin.String(), "BTC") == 0 {
		//	weightTo = Weight{1.2323, nil}
		//	weightFrom = Weight{-3.2323333, nil}
		} else {
			weightTo = Weight{0, nil}
			weightFrom = Weight{0, nil}
		}
		edgeTo := Edge{
			&VertexA,
			&VertexB,
			&weightTo,
		}
		edgeFrom := Edge{
			&VertexB,
			&VertexA,
			&weightFrom,
		}

		if !hasKeyA {
			edgesTo := []Edge{edgeTo}
			graph[VertexA] = edgesTo
		} else {
			graph[VertexA] = append(graph[VertexA], edgeTo)
		}


		if !hasKeyB {
			edgesFrom := []Edge{edgeFrom}
			graph[VertexB] = edgesFrom
		} else {
			graph[VertexB] = append(graph[VertexB], edgeFrom)
		}
	}

	for vertex, edges := range graph {
		var edgesStr = ""
		for _, edge := range edges {
			edgesStr += " *" + edge.To.Coin.String() + " W: " + edge.Weight.String() + "*"
		}
		fmt.Println("Vertex " + vertex.Coin.String() + " | Edges:" + edgesStr)
	}

	fmt.Println("Total pairs: " + strconv.Itoa(len(pairs)))

	//VertexBTC := Vertex{Coin{"BTC"}}
	//edges := graph[VertexBTC]
	//var edgesStr = ""
	//for _, edge := range edges {
	//	edgesStr += " *" + edge.To.Coin.String() + " W: " + edge.Weight.String() + "*"
	//}
	//log.Println("BTC: " + edgesStr)
	//log.Println("len: " + strconv.Itoa(len(edges)))

	log.Println("Running Bellman Ford")
	tStart := time.Now()
	//runBellmanFord()
	delta := time.Since(tStart)
	fmt.Println("Finished in " + strconv.Itoa(int(delta/1000)) + " microseconds")
}

func runBellmanFord() {
	distances = make(map[Vertex]float64)
	parents = make(map[Vertex]*Vertex)

	// Use BTC as a source vertex
	for vertex, _ := range graph {
		if strings.Compare(vertex.Coin.String(), "BTC") == 0 {
			distances[vertex] = 0
		} else {
			distances[vertex] = math.MaxFloat64
			parents[vertex] = nil
		}
	}

	for vertexFrom, edges := range graph {
		for _, edge := range edges {
			vertexTo := *edge.To
			if distances[vertexFrom] + edge.Weight.Value < distances[vertexTo] {
				// relaxation
				distances[vertexTo] = distances[vertexFrom] + edge.Weight.Value
				parents[vertexTo] = &vertexFrom
			}
		}
	}

	for _, edges := range graph {
		for _, edge := range edges {
			vertexFrom := *edge.From
			vertexTo := *edge.To
			if distances[vertexTo] > distances[vertexFrom] + edge.Weight.Value {
				log.Println("Nashel pidora")
			}
		}
	}
}

func printParents() {

}
