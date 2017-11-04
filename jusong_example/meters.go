//
// REST
// ====
// This example demonstrates a HTTP REST web service with some fixture data.
// Follow along the example and patterns.
//
// Also check routes.json for the generated docs from passing the -routes flag
//
// Boot the server:
// ----------------
// $ go run main.go
//
// Client requests:
// ----------------
// $ curl http://localhost:3333/
// root.
//
// $ curl http://localhost:3333/meters
// [{"id":"1","project":"Hi"},{"id":"2","project":"sup"}]
//
// $ curl http://localhost:3333/meters/1
// {"id":"1","project":"Hi"}
//
// $ curl -X DELETE http://localhost:3333/meters/1
// {"id":"1","project":"Hi"}
//
// $ curl http://localhost:3333/meters/1
// "Not Found"
//
// $ curl -X POST -d '{"id":"will-be-omitted","project":"awesomeness"}' http://localhost:3333/meters
// {"id":"97","project":"awesomeness"}
//
// $ curl http://localhost:3333/meters/97
// {"id":"97","project":"awesomeness"}
//
// $ curl http://localhost:3333/meters
// [{"id":"2","project":"sup"},{"id":"97","project":"awesomeness"}]
//
package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

func ListMeters(w http.ResponseWriter, r *http.Request) {
	if err := render.RenderList(w, r, NewMeterListResponse(meters)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// MeterCtx middleware is used to load an Meter object from
// the URL parameters passed through as the request. In case
// the Meter could not be found, we stop here and return a 404.
func MeterCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var meter *Meter
		var err error

		if meterID := chi.URLParam(r, "meterID"); meterID != "" {
			meter, err = dbGetMeter(meterID)
		} else if meterSlug := chi.URLParam(r, "meterSlug"); meterSlug != "" {
			meter, err = dbGetMeterBySlug(meterSlug)
		} else {
			render.Render(w, r, ErrNotFound)
			return
		}
		if err != nil {
			render.Render(w, r, ErrNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), "meter", meter)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SearchMeters searches the Meters data for a matching meter.
// It's just a stub, but you get the idea.
func SearchMeters(w http.ResponseWriter, r *http.Request) {
	render.RenderList(w, r, NewMeterListResponse(meters))
}

// CreateMeter persists the posted Meter and returns it
// back to the client as an acknowledgement.
func CreateMeter(w http.ResponseWriter, r *http.Request) {
	data := &MeterRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	meter := data.Meter
	dbNewMeter(meter)

	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewMeterResponse(meter))
}

// GetMeter returns the specific Meter. You'll notice it just
// fetches the Meter right off the context, as its understood that
// if we made it this far, the Meter must be on the context. In case
// its not due to a bug, then it will panic, and our Recoverer will save us.
func GetMeter(w http.ResponseWriter, r *http.Request) {
	// Assume if we've reach this far, we can access the meter
	// context because this handler is a child of the MeterCtx
	// middleware. The worst case, the recoverer middleware will save us.
	meter := r.Context().Value("meter").(*Meter)

	if err := render.Render(w, r, NewMeterResponse(meter)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// UpdateMeter updates an existing Meter in our persistent store.
func UpdateMeter(w http.ResponseWriter, r *http.Request) {
	meter := r.Context().Value("meter").(*Meter)

	data := &MeterRequest{Meter: meter}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	meter = data.Meter
	dbUpdateMeter(meter.ID, meter)

	render.Render(w, r, NewMeterResponse(meter))
}

// DeleteMeter removes an existing Meter from our persistent store.
func DeleteMeter(w http.ResponseWriter, r *http.Request) {
	var err error

	// Assume if we've reach this far, we can access the meter
	// context because this handler is a child of the MeterCtx
	// middleware. The worst case, the recoverer middleware will save us.
	meter := r.Context().Value("meter").(*Meter)

	meter, err = dbRemoveMeter(meter.ID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	render.Render(w, r, NewMeterResponse(meter))
}

// MeterRequest is the request payload for Meter data model.
//
// NOTE: It's good practice to have well defined request and response payloads
// so you can manage the specific inputs and outputs for clients, and also gives
// you the opportunity to transform data on input or output, for example
// on request, we'd like to protect certain fields and on output perhaps
// we'd like to include a computed field based on other values that aren't
// in the data model. Also, check out this awesome blog post on struct composition:
// http://attilaolah.eu/2014/09/10/json-and-struct-composition-in-go/
type MeterRequest struct {
	*Meter

	User *UserPayload `json:"user,omitempty"`

	ProtectedID string `json:"id"` // override 'id' json to have more control
}

func (a *MeterRequest) Bind(r *http.Request) error {
	// just a post-process after a decode..
	a.ProtectedID = ""                                  // unset the protected ID
	a.Meter.Slug = strings.ToLower(a.Meter.ProjectName) // setup slug Jusong:TODO

	if a.Meter.Duration == "" {
		a.Meter.Duration = "1h"
	}
	var err error
	a.duration, err = time.ParseDuration(a.Meter.Duration)
	if err != nil {
		return errors.Wrapf(err, "Parse %v to duration failed.", a.Duration)
	}

	return nil
}

// MeterResponse is the response payload for the Meter data model.
// See NOTE above in MeterRequest as well.
//
// In the MeterResponse object, first a Render() is called on itself,
// then the next field, and so on, all the way down the tree.
// Render is called in top-down order, like a http handler middleware chain.
type MeterResponse struct {
	*Meter

	// User *UserPayload `json:"user,omitempty"`

	// We add an additional field to the response here.. such as this
	// elapsed computed property
	// Elapsed int64 `json:"elapsed"`
}

func NewMeterResponse(meter *Meter) *MeterResponse {
	resp := &MeterResponse{Meter: meter}

	// if resp.User == nil {
	// 	if user, _ := dbGetUser(resp.UserID); user != nil {
	// 		resp.User = NewUserPayloadResponse(user)
	// 	}
	// }

	return resp
}

func (rd *MeterResponse) Render(w http.ResponseWriter, r *http.Request) error {
	// Pre-processing before a response is marshalled and sent across the wire
	// rd.Elapsed = 10
	return nil
}

type MeterListResponse []*MeterResponse

func NewMeterListResponse(meters []*Meter) []render.Renderer {
	list := []render.Renderer{}
	for _, meter := range meters {
		list = append(list, NewMeterResponse(meter))
	}
	return list
}

//--
// Data model objects and persistence mocks:
//--

// Meter data model. I suggest looking at https://upper.io for an easy
// and powerful data persistence adapter.
type Meter struct {
	ID           string `json:"id"`
	UserID       int64  `json:"user_id"` // the author
	ProjectName  string `json:"project"`
	OraConnectID string `json:"ora_conn"`
	OraUser      string `json:"ora_user"`
	OraPassword  string `json:"ora_password"`
	Duration     string `json:"duration"`
	duration     time.Duration
	Slug         string `json:"slug"`
}

// Meter fixture data
var meters = []*Meter{
	{ID: "1", UserID: 100, ProjectName: "Hi", Slug: "hi"},
	{ID: "2", UserID: 200, ProjectName: "sup", Slug: "sup"},
	{ID: "3", UserID: 300, ProjectName: "alo", Slug: "alo"},
	{ID: "4", UserID: 400, ProjectName: "bonjour", Slug: "bonjour"},
	{ID: "5", UserID: 500, ProjectName: "whats up", Slug: "whats-up"},
}

func dbNewMeter(meter *Meter) (string, error) {
	meter.ID = fmt.Sprintf("%d", rand.Intn(100)+10)
	meters = append(meters, meter)
	return meter.ID, nil
}

func dbGetMeter(id string) (*Meter, error) {
	for _, a := range meters {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, errors.New("meter not found.")
}

func dbGetMeterBySlug(slug string) (*Meter, error) {
	for _, a := range meters {
		if a.Slug == slug {
			return a, nil
		}
	}
	return nil, errors.New("meter not found.")
}

func dbUpdateMeter(id string, meter *Meter) (*Meter, error) {
	for i, a := range meters {
		if a.ID == id {
			meters[i] = meter
			return meter, nil
		}
	}
	return nil, errors.New("meter not found.")
}

func dbRemoveMeter(id string) (*Meter, error) {
	for i, a := range meters {
		if a.ID == id {
			meters = append((meters)[:i], (meters)[i+1:]...)
			return a, nil
		}
	}
	return nil, errors.New("meter not found.")
}
