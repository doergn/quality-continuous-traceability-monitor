package mapping

import (
	"bufio"
	"github.com/SAP/quality-continuous-traceability-monitor/utils"
	"github.com/golang/glog"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AbapParser implements the mapping.Parser interface for ABAP sourcecode
type AbapParser struct {
}

// Parse ABAP sourcecode to seek for traceability comments
func (ap AbapParser) Parse(cfg utils.Config, sc utils.Sourcecode) []TestBacklog {

	var scName string
	if sc.Git.Organization != "" {
		scName = sc.Git.Organization + "/" + sc.Git.Repository
	} else {
		scName = sc.Local
	}

	defer utils.TimeTrack(time.Now(), "Parse ABAP sourcecode ("+scName+")")

	var tb = []TestBacklog{}

	filepath.Walk(sc.Local, func(path string, fi os.FileInfo, err error) error {

		if fi.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".abap" {

			file, err := os.Open(path)
			if err != nil {
				panic(err)
			}
			defer file.Close()

			tb = append(tb, parseJava(file, cfg, sc, file)...)

		}

		return nil
	})

	return tb

}

func parseAbap(coding io.Reader, cfg utils.Config, sc utils.Sourcecode, file *os.File) []TestBacklog {

	var tb = []TestBacklog{}
	var cBli []BacklogItem // Traceability annotation for class
	var mBli []BacklogItem // Traceability annotation for method
	var line,
		cn string
	var err error
	var tm bool // Indicates we've found a "FOR TESTING" annotated method

	reader := bufio.NewReader(coding)
	for {
		line, err = reader.ReadString('\n')

		if err == io.EOF {
			break
		}

		// Empty line
		if line == "" || line == "\n" {
			continue
		}

		// Check whether this line contains the class definition
		if strings.HasPrefix(line, "CLASS ") && strings.Contains(line, " DEFINITION ") {
			// Get the class name
			cns := strings.Index(line, "CLASS ") + 6
			cn = line[cns:]
			cn = cn[:strings.Index(cn, " ")]
			// Don't end loop here, as the CLASS could be FOR TESTING...
		}

		// Is this a test annotation?
		if strings.Contains(line, " FOR TESTING") {
			if strings.HasPrefix(line, "CLASS ") { // We're still in the class definition line -> This is a test class

			} else if cn != "" {
				tm = true
			}
			continue
		}

		// Does the line contain our marker with the backlog item?
		bi := reTraceMarker.FindAllString(line, -1)
		if len(bi) > 0 {
			if cn != "" { // Traceability annotation for a class or a method
				mBli = GetBacklogItem(line) // We're inside a class...must belong to a test method
			} else {
				cBli = GetBacklogItem(line)
			}
			continue
		}

		// Check whether the line contains a test method
		// We're inside a class --> (len(cn) > 0)
		// and we've recently found a traceability annotation --> (m_bli != nil || c_bli != nil)
		// Testing on test annotation (tm) will be done later, as JUnit tests could also be indicated by method name starting with 'test...'
		if len(cn) > 0 && (mBli != nil || cBli != nil) {
			me := strings.LastIndex(line, "{") // Might also be an enum or something else inside a class
			if me != -1 {
				mne := strings.Index(line, "(") // Start of method parameters is end of method name
				if mne != -1 {                  // Might be -1 in case of enums etc.
					mnes := line[:mne] // Get line until end of method name
					mnes = strings.TrimLeft(mnes, " ")
					mns := strings.LastIndex(mnes, " ") // This must be where the method name starts
					m := mnes[mns+1:]

					// We didn't find a test annotation (@Test) yet. Check if method starts with test
					if tm == false && strings.HasPrefix(m, "test") {
						tm = true
					}

					if tm {

						// Create and append test backlog item (for this method)
						t := &Test{getSourcecodeURL(cfg, sc, file), cn, m}
						var tbi TestBacklog
						if cBli != nil {
							tbi = TestBacklog{*t, cBli}
							tb = append(tb, tbi)
						}
						if mBli != nil {
							tbi = TestBacklog{*t, mBli}
							tb = append(tb, tbi)
						}

						// We handled this traceability relevant test method. Reset traceability method annotation
						mBli = nil

						// We handled this test method. Reset @Test annotation marker
						tm = false

					}

				}
			}
		}

	}

	return tb

}
