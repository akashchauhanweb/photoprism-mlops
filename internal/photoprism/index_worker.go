package photoprism

// IndexJob bundles the indexing context for a single media file and its related companions.
type IndexJob struct {
	FileName string
	Related  RelatedFiles
	IndexOpt IndexOptions
	Ind      *Index
}

// IndexWorker consumes IndexJob messages and indexes the related files serially.
// It is intentionally lightweight so the caller can fan out multiple goroutines.
func IndexWorker(jobs <-chan IndexJob) {
	for job := range jobs {
		result := IndexRelated(job.Related, job.Ind, job.IndexOpt)
		if result.Status == IndexAdded && result.PhotoUID != "" {
			originalName := ""
			if job.Related.Main != nil {
				originalName = job.Related.Main.BaseName()
			}
			FireIngestWebhook(result.PhotoUID, originalName, result.FileUID)
		}
	}
}
