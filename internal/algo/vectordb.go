package algo

import (
	"sort"

	"gonum.org/v1/gonum/floats"
)

type DataEntry struct {
	ID    int
	Label string
}

type DataPoint struct {
	ID   int
	Data []float64
}

type VectorDB struct {
	DataPoints []DataPoint
	NumTerms   int
}

func NewVectorDB(nTerms int) *VectorDB {

	db := &VectorDB{
		DataPoints: []DataPoint{},
		NumTerms:   nTerms,
	}

	return db
}

func NewVectorDBFromBinaryStream(s *BinaryFileStream) *VectorDB {

	db := &VectorDB{
		DataPoints: []DataPoint{},
	}

	// Read the number of actions from the file
	numTerms, err := s.ReadInt32()
	if err != nil {
		return nil
	}
	numDatapoints, err := s.ReadInt32()
	if err != nil {
		return nil
	}

	// Read the action vectors from the file
	db.DataPoints = make([]DataPoint, numDatapoints)
	for i := 0; i < int(numDatapoints); i++ {
		id, err := s.ReadInt32()
		if err != nil {
			return nil
		}
		data := make([]float64, numTerms)
		for j := 0; j < int(numTerms); j++ {
			value, err := s.ReadFloat64()
			if err != nil {
				return nil
			}
			data[j] = value
		}
		db.DataPoints[i] = DataPoint{ID: int(id), Data: data}
	}

	return db
}

func (db *VectorDB) GetSimilarEntries(query []float64, minimumScore float64) []int {

	// Calculate the cosine similarity between the query vector and each entry vector
	similarEntries := make([]int, 0, len(db.DataPoints))
	for id, v := range db.DataPoints {
		similarity := CosineSimilarity(query, v.Data)
		if similarity >= minimumScore {
			similarEntries = append(similarEntries, id)
		}
	}

	return similarEntries
}

func (db *VectorDB) GetSimilarEntriesWithScores(query []float64, minimumScore float64, sort bool) map[int]float64 {

	// Calculate the cosine similarity between the query vector and each entry vector
	similarEntries := make(map[int]float64, len(db.DataPoints))
	for id, v := range db.DataPoints {
		similarity := CosineSimilarity(query, v.Data)
		if similarity > minimumScore {
			similarEntries[id] = similarity
		}
	}

	if sort {
		return sortMapByValue(similarEntries)
	}

	return similarEntries
}

func (db *VectorDB) GetDataPoint(id int) DataPoint {
	return db.DataPoints[id]
}

func (db *VectorDB) SaveToBinaryStream(s *BinaryFileStream) error {

	// Write the number of data points to the file
	if err := s.WriteInt32(int32(len(db.DataPoints))); err != nil {
		return err
	}
	// Write the action vectors to the file
	for _, v := range db.DataPoints {
		if err := s.WriteInt32(int32(v.ID)); err != nil {
			return err
		}
		if err := s.WriteInt32(int32(len(v.Data))); err != nil {
			return err
		}
		for _, value := range v.Data {
			if err := s.WriteFloat64(value); err != nil {
				return err
			}
		}
	}

	return nil
}

// Calculate the cosine similarity between two vectors
func CosineSimilarity(vector1, vector2 []float64) float64 {
	dotProduct := floats.Dot(vector1, vector2)
	magnitude1 := floats.Norm(vector1, 2)
	magnitude2 := floats.Norm(vector2, 2)
	if magnitude1 == 0 || magnitude2 == 0 {
		return 0
	}
	return dotProduct / (magnitude1 * magnitude2)
}

// Sort a map by its values in descending order
func sortMapByValue(m map[int]float64) map[int]float64 {
	type kv struct {
		Key   int
		Value float64
	}
	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})
	sortedMap := make(map[int]float64)
	for _, kv := range sorted {
		sortedMap[kv.Key] = kv.Value
	}
	return sortedMap
}
