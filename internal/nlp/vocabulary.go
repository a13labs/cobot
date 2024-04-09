package nlp

import (
	"sort"
	"strings"

	"github.com/a13labs/cobot/internal/algo"
	"github.com/a13labs/cobot/internal/io"
	"github.com/kljensen/snowball"
)

type Term struct {
	Token string
	IDF   float64
}

type Vocabulary struct {
	Terms    []Term
	Language string
}

func NewVocabulary(docs algo.StringList, language string) *Vocabulary {

	v := &Vocabulary{
		Language: language,
		Terms:    []Term{},
	}

	// Create a vocabulary of all the terms in the dataset
	vocabulary := map[string]struct{}{}
	for _, d := range docs {
		content := strings.ToLower(d)
		for _, token := range v.Tokenize(content) {
			vocabulary[token] = struct{}{}
		}
	}

	// Order the vocabulary alphabetically
	terms := make(algo.StringList, 0, len(vocabulary))
	for key := range vocabulary {
		terms = append(terms, key)
	}
	sort.Strings(terms)

	// Calculate the IDF values for each term
	v.Terms = make([]Term, len(terms))
	for i, term := range terms {
		freq := 0
		for _, d := range docs {
			content := strings.ToLower(d)
			if strings.Contains(content, term) {
				freq++
			}
		}
		idf := 0.0
		if freq != 0 {
			idf = float64(len(docs)) / float64(freq)
		}
		v.Terms[i] = Term{Token: term, IDF: idf}
	}

	return v
}

func NewVocabularyFromBinaryStream(s *io.BinaryFileStream, language string) *Vocabulary {
	v := &Vocabulary{
		Language: language,
		Terms:    []Term{},
	}

	// Read the number of terms from the file
	numTerms, err := s.ReadInt32()
	if err != nil {
		return nil
	}
	// Read the term data from the file
	v.Terms = make([]Term, numTerms)
	for i := 0; i < int(numTerms); i++ {
		length, err := s.ReadInt32()
		if err != nil {
			return nil
		}
		term := make([]byte, length)
		_, err = s.Read(term)
		if err != nil {
			return nil
		}
		idf, err := s.ReadFloat64()
		if err != nil {
			return nil
		}
		v.Terms[i] = Term{Token: string(term), IDF: idf}
	}
	return v
}

// Tokenize the text and stem the tokens
func (v *Vocabulary) Tokenize(text string) []string {
	tokens := strings.Fields(text)
	stemmedTokens := make([]string, len(tokens))
	for i, token := range tokens {
		stemmedToken, _ := snowball.Stem(token, v.Language, false)
		stemmedTokens[i] = stemmedToken
	}
	return stemmedTokens
}

func (v *Vocabulary) CalculateTFIDFVector(tokens []string) []float64 {

	// Create a TF-IDF vector
	vector := make([]float64, len(v.Terms))

	// Calculate the TF-IDF values for each term
	tokensStr := strings.ToLower(strings.Join(tokens, " "))
	for i, term := range v.Terms {

		tf := float64(strings.Count(tokensStr, term.Token))
		idf := term.IDF
		vector[i] = tf * idf
	}

	return vector
}

func (v *Vocabulary) SaveToBinaryStream(s *io.BinaryFileStream) error {
	// Write the number of terms to the file
	err := s.WriteInt32(int32(len(v.Terms)))
	if err != nil {
		return err
	}
	// Write the term data to the file
	for _, term := range v.Terms {
		err = s.WriteInt32(int32(len(term.Token)))
		if err != nil {
			return err
		}
		_, err = s.WriteString(term.Token)
		if err != nil {
			return err
		}
		err = s.WriteFloat64(term.IDF)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *Vocabulary) GetTerms() []Term {
	return v.Terms
}
