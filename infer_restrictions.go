package gist

import (
	"errors"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/dominikbraun/graph"
	"strings"
)

type tableId string

type table struct {
	Name tableId `csv:"# Name"`
	// ignore other fields for now
}

type association struct {
	TableA        tableId `csv:"# Table A"`
	TableB        tableId `csv:"Table B"`
	FirstInsert   string  `csv:"First-insert"`
	Cardinality   string  `csv:"Cardinality (opt)"`
	JoinCondition string  `csv:"Join-condition"`
	Name          string  `csv:"Name"`
	Author        string  `csv:"Author"`
}

func InferRestrictions(tables []*table, associations []*association, subjectTables []tableId) ([]string, []string) {

	g := buildGraph(tables, associations)

	/* extract relevant data structures */
	adjacencyMap, err := g.AdjacencyMap()
	if err != nil {
		panic(err) //todo
	}
	predecessorMap, err := g.PredecessorMap()
	if err != nil {
		panic(err) //todo
	}

	/* actual algorithm */
	frontier := mapset.NewSet[tableId]()
	visited := mapset.NewSet[tableId]()
	enabledAssociations := mapset.NewSet[string]()
	disabledAssociations := mapset.NewSet[string]()

	// Expand all the tables *directly* associated with the subjectTables.
	// Even the child tables (tables that reference a subject table) are expanded here
	// since we want all the records that belong to the subject tables.
	for _, subjectTable := range subjectTables {
		for parent, edge := range adjacencyMap[subjectTable] {
			frontier.Add(parent)
			associationName := edge.Properties.Attributes["name"]
			enabledAssociations.Add(associationName)
			disabledAssociations.Add(inverse(associationName))
		}

		for child, edge := range predecessorMap[subjectTable] {
			frontier.Add(child)
			associationName := edge.Properties.Attributes["name"]
			if !enabledAssociations.Contains(associationName) {
				enabledAssociations.Add(inverse(associationName))
				disabledAssociations.Add(associationName)
			}
		}
		visited.Add(subjectTable)
	}

	for {
		unexploredEdges := make(map[string]graph.Edge[tableId], 10)

		// transitively follow fks until there are no more fks to follow
		for !frontier.IsEmpty() {
			table, _ := frontier.Pop()

			if visited.Contains(table) {
				continue
			}

			// expand parent tables (follow fks)
			for parent, edge := range adjacencyMap[table] {
				associationName := edge.Properties.Attributes["name"]
				if !enabledAssociations.Contains(inverse(associationName)) {
					enabledAssociations.Add(associationName)
					disabledAssociations.Add(inverse(associationName))
				}
				if !visited.Contains(parent) {
					frontier.Add(parent)
				}
			}

			// record unexplored inverse fks
			for _, edge := range predecessorMap[table] {
				associationName := edge.Properties.Attributes["name"]
				if !enabledAssociations.Contains(associationName) {
					unexploredEdges[associationName] = edge
				}
			}

			visited.Add(table)
		}

		// expand unexplored tables
		for assoc, e := range unexploredEdges {
			if !enabledAssociations.Contains(assoc) {
				enabledAssociations.Add(inverse(assoc))
				disabledAssociations.Add(assoc)
			}
			if !visited.Contains(e.Source) {
				frontier.Add(e.Source)
			}
		}

		// If no more tables were added to the frontier after expanding unexplored tables, then
		// it means there are no more reachable tables and thus there is nothing more to be done
		if frontier.IsEmpty() {
			break
		}
	}
	return enabledAssociations.ToSlice(), disabledAssociations.ToSlice()
}

func buildGraph(tables []*table, associations []*association) graph.Graph[tableId, *table] {
	tableIdentity := func(t *table) tableId {
		return t.Name
	}
	g := graph.New[tableId, *table](tableIdentity, graph.Directed())
	for _, table := range tables {
		err := g.AddVertex(table)
		if err != nil {
			panic(err) //todo
		}
	}
	for _, association := range associations {
		err := g.AddEdge(association.TableA, association.TableB, graph.EdgeAttribute("name", association.Name))
		if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
			panic(err) //todo
		}
	}
	return g
}

func inverse(associationName string) string {
	if strings.HasPrefix(associationName, "inverse-") {
		return strings.TrimPrefix(associationName, "inverse-")
	}
	return "inverse-" + associationName
}
