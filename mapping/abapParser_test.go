package mapping

import (
	"github.com/SAP/quality-continuous-traceability-monitor/utils"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestBacklog mappings (correct)
var testAbapCode = []testMapping{
	{input: `
CLASS ltcl_test DEFINITION FOR TESTING
    DURATION SHORT
    RISK LEVEL HARMLESS
    FINAL.

  PRIVATE SECTION.

* Trace(Jira:MYJIRAPROJECT-72)
    METHODS: test FOR TESTING.

ENDCLASS.       "ltcl_Test

CLASS ltcl_test IMPLEMENTATION.

  METHOD test.

    DATA: lo_category TYPE REF TO zcl_aoc_category.

* just test that it does not dump

    CREATE OBJECT lo_category.

  ENDMETHOD.

ENDCLASS.
	`,
		expectedResult: []TestBacklog{{Test: Test{ClassName: "com.sap.ctm.testing.MyTest", FileURL: "testFile.abap", Method: "test"},
			BacklogItem: []BacklogItem{{ID: "MYJIRAPROJECT-72", Source: Jira}}}}}}

func TestAbapParsing(t *testing.T) {

	cfg := new(utils.Config)
	cfg.Mapping.Local = "NonPersistedMappingFileForTesting"
	cfg.Github.BaseURL = "https://github.com"

	var sc = utils.Sourcecode{Git: utils.Git{Branch: "master", Organization: "testOrg", Repository: "testRepo"}, Language: "abap", Local: "./"}
	var file = os.NewFile(0, "testFile.abap")

	for i, mapping := range testAbapCode {
		tb := parseAbap(strings.NewReader(mapping.input), *cfg, sc, file)
		if !compareTestBacklog(tb, mapping.expectedResult) {
			t.Error("Comparism of ABAP Code (No. " + strconv.Itoa(i) + "): \n" + mapping.input + "\n with expected result failed.")
		}
	}

}
