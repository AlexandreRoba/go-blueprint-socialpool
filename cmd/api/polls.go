package main

import (
	"gopkg.in/mgo.v2/bson"
	"net/http"
	"errors"
	"gopkg.in/mgo.v2"
)

type poll struct {
	ID      bson.ObjectId `bson:"_id" json:"id"`
	Title   string `json:"title"`
	Options []string `json:"options"`
	Results map[string]int `json:"results,omitempty"`
	APIKey  string `json:"apikey"`
}

func (s *Server) handlePolls(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handlePollsGet(w, r)
		return
	case "POST":
		s.handlePollsPost(w, r)
		return
	case "DELETE":
		s.handlePollsDelete(w, r)
		return
	}
	//Not found
	respondHTTPErr(w, r, http.StatusNotFound)
}

func (s *Server) handlePollsGet(w http.ResponseWriter, r *http.Request) {
	session := s.db.Copy()
	defer session.Close()
	c := session.DB("ballots").C("polls")
	var q *mgo.Query
	p := NewPath(r.URL.Path)
	if p.HasID() {
		//Get specific poll
		q = c.FindId(bson.ObjectIdHex(p.ID))
	} else {
		// get all polls
		q = c.Find(nil)
	}
	var result []*poll
	if err := q.All(&result); err != nil {
		respondErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond(w, r, http.StatusOK, &result)
}

func (s *Server) handlePollsPost(w http.ResponseWriter, r *http.Request) {
	session := s.db.Copy()
	defer session.Close()
	c := session.DB("ballots").C("polls")
	var p poll
	if err := decodeBody(r, &p); err != nil {
		respondErr(w, r, http.StatusBadRequest, "failed to read poll from request", err)
		return
	}
	apikey, ok := APIKey(r.Context())
	if ok {
		p.APIKey = apikey
	}
	p.ID = bson.NewObjectId()
	if err := c.Insert(p); err != nil {
		respondErr(w, r, http.StatusInternalServerError, "failed to insert poll", err)
		return
	}
	w.Header().Set("Location", "polls/"+p.ID.Hex())
	respond(w, r, http.StatusCreated, nil)
}

func (s *Server) handlePollsDelete(w http.ResponseWriter, r *http.Request) {

}
