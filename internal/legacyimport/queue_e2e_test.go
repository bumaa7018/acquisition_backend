//go:build e2e

package legacyimport

// waitIdle — өмнө нь дараалалд орсон бүх ажил дуусахыг хүлээнэ (тестэд).
func (s *Service) waitIdle() {
	done := make(chan struct{})
	s.queue <- func() { close(done) }
	<-done
}
