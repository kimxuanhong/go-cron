package cron

// Scheduler là interface bọc các phương thức của cron để quản lý cron jobs.
type Scheduler interface {
	AddJob(cronExpr string, jobFunc func()) error
	SetDirs(dirs ...string)
	RegisterHandlers(handlers ...interface{})
	ParseSourceForCronJobs()
	Start() error
	Stop()
}
