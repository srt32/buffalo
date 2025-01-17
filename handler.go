package buffalo

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// Handler is the basis for all of Buffalo. A Handler
// will be given a Context interface that represents the
// give request/response. It is the responsibility of the
// Handler to handle the request/response correctly. This
// could mean rendering a template, JSON, etc... or it could
// mean returning an error.
/*
	func (c Context) error {
		return c.Render(200, render.String("Hello World!"))
	}

	func (c Context) error {
		return c.Redirect(301, "http://github.com/gobuffalo/buffalo")
	}

	func (c Context) error {
		return c.Error(422, errors.New("oops!!"))
	}
*/
type Handler func(Context) error

func (a *App) newContext(info RouteInfo, res http.ResponseWriter, req *http.Request) Context {
	ws := res.(*buffaloResponse)
	params := req.URL.Query()
	vars := mux.Vars(req)
	for k, v := range vars {
		params.Set(k, v)
	}

	session := a.getSession(req, ws)

	return &DefaultContext{
		Context:  req.Context(),
		response: ws,
		request:  req,
		params:   params,
		logger:   a.Logger,
		session:  session,
		flash:    newFlash(session),
		data: map[string]interface{}{
			"env":           a.Env,
			"routes":        a.Routes(),
			"current_route": info,
		},
	}
}

func (a *App) handlerToHandler(info RouteInfo, h Handler) http.Handler {
	hf := func(res http.ResponseWriter, req *http.Request) {
		c := a.newContext(info, res, req)

		defer c.Flash().persist(c.Session())

		err := a.Middleware.handler(h)(c)

		if err != nil {
			status := 500
			// unpack root cause and check for HTTPError
			cause := errors.Cause(err)
			httpError, ok := cause.(HTTPError)
			if ok {
				status = httpError.Status
			}
			eh := a.ErrorHandlers.Get(status)
			err = eh(status, err, c)
			if err != nil {
				// things have really hit the fan if we're here!!
				a.Logger.Error(err)
				c.Response().WriteHeader(500)
				c.Response().Write([]byte(err.Error()))
			}
			// err := c.Error(500, err)
			// a.Logger.Error(err)
		}
	}
	return http.HandlerFunc(hf)
}
