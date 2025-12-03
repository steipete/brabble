package run

import "context"

func (s *Server) hookWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-s.hookCh:
			if err := s.hook.Run(ctx, job); err != nil {
				s.logger.Errorf("hook: %v", err)
				continue
			}
			s.metrics.incSent()
		}
	}
}
