package reporter

//func TestJunitConverter(t *testing.T) {
//	reportCollectionRaw, err := os.ReadFile("./testdata/report-debian.json")
//	require.NoError(t, err)
//
//	yr := &policy.ReportCollection{}
//	err = json.Unmarshal(reportCollectionRaw, yr)
//	require.NoError(t, err)
//
//	buf := bytes.Buffer{}
//	writer := shared.IOWriter{Writer: &buf}
//	err = ReportCollectionToJunit(yr, &writer)
//	require.NoError(t, err)
//
//	assert.Contains(t, buf.String(), `<failure message="results do not match" type="fail"></failure>`)
//}
