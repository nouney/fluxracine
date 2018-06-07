package chat

// Opt is a function used to configure the Server object
type Opt = func(c *Server) error

// WithHTTPAddress sets the address on which the internal HTTP server
// will listen.
func WithHTTPAddress(a string) Opt {
	return func(s *Server) error {
		s.httpAddr = a
		return nil
	}
}
