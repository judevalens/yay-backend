package job_queue

import (
	"fmt"
	"time"
)

type QueueWorker interface {
	 Worker(id int,job interface{})
}

type WorkerPool struct {
	queueSize int
	nWorker  int
	JobQueue chan interface{}
	queue    []interface{}
	Worker   QueueWorker

}

type job struct {
	data interface{}
}


func NewWorkerPool(worker QueueWorker,nWorker, queueSize int,) *WorkerPool {
	workerPool := new(WorkerPool)
	workerPool.JobQueue = make(chan interface{},queueSize)
	workerPool.nWorker = nWorker
	workerPool.Worker = worker
	workerPool.queueSize = queueSize
	return workerPool
}


/*func worker(workerPool *WorkerPool, id int ){
	for j := range  workerPool.jobQueue{
		ans := j * 5
		fmt.Printf("job done by worker #%v, result is : %v\n",id, ans)
	}
}
*/
func (workerPool *WorkerPool)Start(){
	for k:= 0; k < workerPool.nWorker; k++ {
		fmt.Printf("launched worker #%v\n",k)
		go func() {
			for {
				job := <- workerPool.JobQueue
				workerPool.Worker.Worker(k,job)
			}
		}()
	}
}



func (workerPool *WorkerPool) AddJob(j interface{})bool{
	var res bool
	select {
	case workerPool.JobQueue <- j:
		res=  true
	case <- time.After(time.Second*5):
		res =  false
	}
	return res
}