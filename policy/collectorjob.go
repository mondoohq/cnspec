package policy

import "github.com/rs/zerolog"

type InvalidCollectorJobError struct {
	InvalidSpecsByReportingJob     map[string][]string
	InvalidNotifiersByReportingJob map[string][]string
}

func newInvalidCollectorJobError() *InvalidCollectorJobError {
	return &InvalidCollectorJobError{
		InvalidSpecsByReportingJob:     make(map[string][]string),
		InvalidNotifiersByReportingJob: make(map[string][]string),
	}
}

func (e *InvalidCollectorJobError) addInvalidSpec(jobUUID string, specRef string) {
	e.InvalidSpecsByReportingJob[jobUUID] = append(e.InvalidSpecsByReportingJob[jobUUID], specRef)
}

func (e *InvalidCollectorJobError) addInvalidNotifier(jobUUID string, jobRef string) {
	e.InvalidNotifiersByReportingJob[jobUUID] = append(e.InvalidNotifiersByReportingJob[jobUUID], jobRef)
}

func (e *InvalidCollectorJobError) Error() string {
	return "invalid collector job"
}

func (e *InvalidCollectorJobError) MarshalZerologObject(ev *zerolog.Event) {
	specsByJob := zerolog.Dict()
	for rjUUID, specRefs := range e.InvalidSpecsByReportingJob {
		specsByJob.Strs(rjUUID, specRefs)
	}
	ev.Dict("specs-by-job", specsByJob)

	notifiersByJob := zerolog.Dict()
	for rjUUID, notifierRef := range e.InvalidNotifiersByReportingJob {
		notifiersByJob.Strs(rjUUID, notifierRef)
	}
	ev.Dict("notifiers-by-job", notifiersByJob)
}

// Validate that this collector job is wired correctly and adheres to internal consistency
func (c *CollectorJob) Validate() error {
	invalid := false
	invalidCollectorJobError := newInvalidCollectorJobError()

	for _, job := range c.ReportingJobs {
		for ref := range job.ChildJobs {
			if _, ok := c.ReportingJobs[ref]; !ok {
				invalid = true
				invalidCollectorJobError.addInvalidSpec(job.Uuid, ref)
			}
		}

		for ref := range job.Notify {
			ref := job.Notify[ref]
			if _, ok := c.ReportingJobs[ref]; !ok {
				invalid = true
				invalidCollectorJobError.addInvalidNotifier(job.Uuid, ref)
			}
		}
	}

	if !invalid {
		return nil
	}

	return invalidCollectorJobError
}
