package core

import (
	"context"
	"encoding/json"
	"time"

	"github.com/blugelabs/bluge"
	v1 "github.com/prabhatsharma/zinc/pkg/meta/v1"
	"github.com/prabhatsharma/zinc/pkg/startup"
	"github.com/prabhatsharma/zinc/pkg/uquery"
	"github.com/rs/zerolog/log"
)

func (index *Index) Search(iQuery *v1.ZincQuery) (v1.SearchResponse, error) {
	var Hits []v1.Hit

	var searchRequest bluge.SearchRequest
	if iQuery.MaxResults > startup.MAX_RESULTS {
		iQuery.MaxResults = startup.MAX_RESULTS
	}

	var err error
	switch iQuery.SearchType {
	case "alldocuments":
		searchRequest, err = uquery.AllDocuments(iQuery)
	case "wildcard":
		searchRequest, err = uquery.WildcardQuery(iQuery)
	case "fuzzy":
		searchRequest, err = uquery.FuzzyQuery(iQuery)
	case "term":
		searchRequest, err = uquery.TermQuery(iQuery)
	case "daterange":
		searchRequest, err = uquery.DateRangeQuery(iQuery)
	case "matchall":
		searchRequest, err = uquery.MatchAllQuery(iQuery)
	case "match":
		searchRequest, err = uquery.MatchQuery(iQuery)
	case "matchphrase":
		searchRequest, err = uquery.MatchPhraseQuery(iQuery)
	case "multiphrase":
		searchRequest, err = uquery.MultiPhraseQuery(iQuery)
	case "prefix":
		searchRequest, err = uquery.PrefixQuery(iQuery)
	case "querystring":
		searchRequest, err = uquery.QueryStringQuery(iQuery)
	default:
		// default use alldocuments search
		searchRequest, err = uquery.AllDocuments(iQuery)
	}

	if err != nil {
		return v1.SearchResponse{
			Error: err.Error(),
		}, err
	}

	// handle aggregations
	mapping, _ := index.GetStoredMapping()
	err = uquery.AddAggregations(searchRequest, iQuery.Aggregations, mapping)
	if err != nil {
		return v1.SearchResponse{
			Error: err.Error(),
		}, err
	}

	reader, err := index.Writer.Reader()
	if err != nil {
		log.Printf("error accessing reader: %v", err)
	}
	defer reader.Close()

	dmi, err := reader.Search(context.Background(), searchRequest)
	if err != nil {
		log.Printf("error executing search: %v", err)
	}

	// highlighter := highlight.NewANSIHighlighter()

	next, err := dmi.Next()
	for err == nil && next != nil {
		var result map[string]interface{}
		var id string
		var timestamp time.Time
		err = next.VisitStoredFields(func(field string, value []byte) bool {
			if field == "_source" {
				json.Unmarshal(value, &result)
				return true
			} else if field == "_id" {
				id = string(value)
				return true
			} else if field == "@timestamp" {
				timestamp, _ = bluge.DecodeDateTime(value)
				return true
			}
			return true
		})
		if err != nil {
			log.Printf("error accessing stored fields: %v", err)
		}

		hit := v1.Hit{
			Index:     index.Name,
			Type:      index.Name,
			ID:        id,
			Score:     next.Score,
			Timestamp: timestamp,
			Source:    result,
		}
		Hits = append(Hits, hit)

		next, err = dmi.Next()
	}
	if err != nil {
		log.Printf("error iterating results: %v", err)
	}

	resp := v1.SearchResponse{
		Took: int(dmi.Aggregations().Duration().Milliseconds()),
		Hits: v1.Hits{
			Total: v1.Total{
				Value: int(dmi.Aggregations().Count()),
			},
			MaxScore: dmi.Aggregations().Metric("max_score"),
			Hits:     Hits,
		},
	}

	if len(iQuery.Aggregations) > 0 {
		resp.Aggregations, err = uquery.ParseAggregations(dmi.Aggregations())
		if err != nil {
			log.Printf("error parse aggregation results: %v", err)
		}
		if len(resp.Aggregations) > 0 {
			delete(resp.Aggregations, "count")
			delete(resp.Aggregations, "duration")
			delete(resp.Aggregations, "max_score")
		}
	}

	return resp, nil
}
