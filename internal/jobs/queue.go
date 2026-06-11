package jobs

import "log"

type Job func()

type Queue struct {
	jobs chan Job
}

func NewQueue(buffer int) *Queue {
	return &Queue{jobs: make(chan Job, buffer)}
}

func (q *Queue) Start(workers int) {
	if workers < 1 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		go func() {
			for job := range q.jobs {
				func() {
					defer func() {
						if recovered := recover(); recovered != nil {
							log.Println("background job recovered:", recovered)
						}
					}()
					job()
				}()
			}
		}()
	}
}

func (q *Queue) Enqueue(job Job) {
	select {
	case q.jobs <- job:
	default:
		go job()
	}
}

var Default = NewQueue(128)
