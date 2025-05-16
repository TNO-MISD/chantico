package prometheus

import (
	"encoding/json"
)

type PrometheusRequestResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				Name             string `json:"__name__"`
				Instance         string `json:"instance"`
				Job              string `json:"job"`
				SdbDevIDIndex    string `json:"sdbDevIdIndex"`
				SdbDevOutMtIndex string `json:"sdbDevOutMtIndex"`
			} `json:"metric"`
			Values [][]string `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

func (requestResponse *PrometheusRequestResponse) Parse(data []byte) error {
	var err error

	err = json.Unmarshal(data, &requestResponse)
	if err != nil {
		return err
	}
	return nil
}
