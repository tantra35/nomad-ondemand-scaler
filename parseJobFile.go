package main

import (
	"fmt"
	"os"

	nomadapi "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec"
	"github.com/hashicorp/nomad/jobspec2"
)

func parseJobFile(_jobfilePath string) (*nomadapi.Job, error) {
	jobfile, lerr := os.Open(_jobfilePath)
	if lerr != nil {
		return nil, fmt.Errorf("can't read nomad job file due: %s", lerr)
	}
	defer jobfile.Close()
	njob, lerr := jobspec.Parse(jobfile)
	if lerr != nil {
		jobfile.Seek(0, os.SEEK_SET)
		njob, lerr = jobspec2.Parse(_jobfilePath, jobfile)
		if lerr != nil {
			return nil, fmt.Errorf("can't parse nomad job file after fallback due: %s", lerr)
		}
	}

	njob.Canonicalize()
	return njob, nil
}
