package rolling

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	backupTimeFormat = "2006-01-02T15-04-05.000"
)

// ensure we always implement io.WriteCloser
var _ io.WriteCloser = (*Roller)(nil)

var (
	// currentTime exists so it can be mocked out by tests.
	currentTime = time.Now
)

// ErrWriteTooLong indicates that a single write that is longer than the max
// size allowed in a single file.
var ErrWriteTooLong = errors.New("write exceeds max file length")

// Roller is an io.WriteCloser that writes to the specified filename.
//
// Roller opens or creates the logfile on first Write.  If the file exists and
// is less than maxSize megabytes, lumberjack will open and append to that file.
// If the file exists and its size is >= maxSize megabytes, the file is renamed
// by putting the current time in a timestamp in the name immediately before the
// file's extension (or the end of the filename if there's no extension). A new
// log file is then created using original filename.
//
// Whenever write would cause the current log file exceed maxSize megabytes,
// the current file is closed, renamed, and a new log file created with the
// original name. Thus, the filename you give Roller is always the "current" log
// file.
//
// Backups use the log file name given to Roller, in the form
// `name-timestamp.ext` where name is the filename without the extension,
// timestamp is the time at which the log was rotated formatted with the
// time.Time format of `2006-01-02T15-04-05.000` and the extension is the
// original extension.  For example, if your Roller.filename is
// `/var/log/foo/server.log`, a backup created at 6:30pm on Nov 11 2016 would
// use the filename `/var/log/foo/server.log.2016-11-04T18-30-00.000`
//
// # Cleaning Up Old Log Files
//
// Whenever a new logfile gets created, old log files may be deleted.  The most
// recent files according to the encoded timestamp will be retained, up to a
// number equal to maxBackups (or all of them if maxBackups is 0).  Any files
// with an encoded timestamp older than maxAge days are deleted, regardless of
// maxBackups.  Note that the time encoded in the timestamp is the rotation
// time, which may differ from the last time that file was written to.
//
// If maxBackups and maxAge are both 0, no old log files will be deleted.
type Roller struct {
	// filename is the file to write logs to.  Backup log files will be retained
	// in the same directory.
	filename string

	// maxSize is the maximum size in bytes rotated.
	maxSize int64

	// maxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	maxAge int

	// maxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though maxAge may still cause them to get
	// deleted.)
	maxBackups int

	size int64
	file *os.File
	mu   sync.Mutex

	millCh    chan bool
	startMill sync.Once
}

type Option func(roller *Roller)

func WithMaxAge(age int) Option {
	return func(roller *Roller) {
		roller.maxAge = age
	}
}

func WithMaxBackups(backups int) Option {
	return func(roller *Roller) {
		roller.maxBackups = backups
	}
}

func NewRoller(filename string, maxSize int64, opt ...Option) (*Roller, error) {
	if maxSize <= 0 {
		return nil, errors.New("max size cannot be 0")
	}
	if filename == "" {
		return nil, errors.New("filename cannot be empty")
	}

	r := &Roller{
		filename:   filename,
		maxSize:    maxSize,
		maxBackups: 30,
		maxAge:     30,
	}

	for _, o := range opt {
		o(r)
	}

	err := r.openExistingOrNew(0)
	if err != nil {
		return nil, fmt.Errorf("can't open file: %w", err)
	}
	return r, nil
}

// Write implements io.Writer.  If a write would cause the log file to be larger
// than maxSize, the file is closed, renamed to include a timestamp of the
// current time, and a new log file is created using the original log file name.
// If the length of the write is greater than maxSize, an error is returned.
func (r *Roller) Write(p []byte) (n int, err error) {
	writeLen := int64(len(p))
	if writeLen > r.maxSize {
		return 0, fmt.Errorf(
			"write length %d, max size %d: %w", writeLen, r.maxSize, ErrWriteTooLong,
		)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size+writeLen > r.maxSize {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = r.file.Write(p)
	r.size += int64(n)

	return n, err
}

// Close implements io.Closer, and closes the current logfile.
func (r *Roller) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.close()
}

// close closes the file if it is open.
func (r *Roller) close() error {
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return err
}

// Rotate causes Roller to close the existing log file and immediately create a
// new one.  This is a helper function for applications that want to initiate
// rotations outside of the normal rotation rules, such as in response to
// SIGHUP.  After rotating, this initiates compression and removal of old log
// files according to the configuration.
func (r *Roller) Rotate() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rotate()
}

// rotate closes the current file, moves it aside with a timestamp in the name,
// (if it exists), opens a new file with the original filename, and then runs
// post-rotation processing and removal.
func (r *Roller) rotate() error {
	if err := r.close(); err != nil {
		return err
	}
	if err := r.openNew(); err != nil {
		return err
	}
	r.mill()
	return nil
}

// openNew opens a new log file for writing, moving any old log file out of the
// way.  This methods assumes the file has already been closed.
func (r *Roller) openNew() error {
	err := os.MkdirAll(r.dir(), 0755)
	if err != nil {
		return fmt.Errorf("can't make directories for new logfile: %s", err)
	}

	name := r.filename
	mode := os.FileMode(0600)
	info, err := os.Stat(name)
	if err == nil {
		// Copy the mode off the old logfile.
		mode = info.Mode()
		// move the existing file
		newname := backupName(name)
		if err := os.Rename(name, newname); err != nil {
			return fmt.Errorf("can't rename log file: %s", err)
		}
	}

	// we use truncate here because this should only get called when we've moved
	// the file ourselves. if someone else creates the file in the meantime,
	// just wipe out the contents.
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("can't open new logfile: %s", err)
	}
	r.file = f
	r.size = 0
	return nil
}

// backupName creates a new filename from the given name
func backupName(name string) string {
	return fmt.Sprintf("%s.%s", name, currentTime().Format(backupTimeFormat))
}

// openExistingOrNew opens the logfile if it exists and if the current write
// would not put it over maxSize.  If there is no such file or the write would
// put it over the maxSize, a new file is created.
func (r *Roller) openExistingOrNew(writeLen int) error {
	r.mill()

	info, err := os.Stat(r.filename)
	if os.IsNotExist(err) {
		return r.openNew()
	}
	if err != nil {
		return fmt.Errorf("error getting log file info: %s", err)
	}

	if info.Size()+int64(writeLen) >= r.maxSize {
		return r.rotate()
	}

	file, err := os.OpenFile(r.filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// if we fail to open the old log file for some reason, just ignore
		// it and open a new log file.
		return r.openNew()
	}
	r.file = file
	r.size = info.Size()
	return nil
}

// millRunOnce performs compression and removal of stale log files.
// Log files are compressed if enabled via configuration and old log
// files are removed, keeping at most l.MaxBackups files, as long as
// none of them are older than maxAge.
func (r *Roller) millRunOnce() error {
	if r.maxBackups == 0 && r.maxAge == 0 {
		return nil
	}

	files, err := r.oldLogFiles()
	if err != nil {
		return err
	}

	var remove []logInfo

	if r.maxBackups > 0 && r.maxBackups < len(files) {
		preserved := make(map[string]bool)
		for _, f := range files {
			fn := f.Name()
			preserved[fn] = true

			if len(preserved) > r.maxBackups {
				remove = append(remove, f)
			}
		}
	}
	if r.maxAge > 0 {
		diff := time.Duration(int64(24*time.Hour) * int64(r.maxAge))
		cutoff := currentTime().Add(-1 * diff)
		for _, f := range files {
			if f.timestamp.Before(cutoff) {
				remove = append(remove, f)
			}
		}
	}

	for _, f := range remove {
		errRemove := os.Remove(filepath.Join(r.dir(), f.Name()))
		if err == nil && errRemove != nil {
			err = errRemove
		}
	}
	return err
}

// millRun runs in a goroutine to manage post-rotation compression and removal
// of old log files.
func (r *Roller) millRun() {
	for range r.millCh {
		// what am I going to do, log this?
		_ = r.millRunOnce()
	}
}

// mill performs post-rotation compression and removal of stale log files,
// starting the mill goroutine if necessary.
func (r *Roller) mill() {
	r.startMill.Do(func() {
		r.millCh = make(chan bool, 1)
		go r.millRun()
	})
	select {
	case r.millCh <- true:
	default:
	}
}

// oldLogFiles returns the list of backup log files stored in the same
// directory as the current log file, sorted by ModTime
func (r *Roller) oldLogFiles() ([]logInfo, error) {
	files, err := os.ReadDir(r.dir())
	if err != nil {
		return nil, fmt.Errorf("can't read log file directory: %s", err)
	}
	var logFiles []logInfo

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}

		if t, err := r.timeFromName(f.Name()); err == nil {
			logFiles = append(logFiles, logInfo{t, info})
			continue
		}
		// error parsing means that the suffix at the end was not generated
		// by lumberjack, and therefore it's not a backup file.
	}

	sort.Sort(byFormatTime(logFiles))

	return logFiles, nil
}

// timeFromName extracts the formatted time from the filename by stripping off
// the filename's prefix and extension. This prevents someone's filename from
// confusing time.parse.
func (r *Roller) timeFromName(filename string) (time.Time, error) {
	name := filepath.Base(r.filename)
	ts := strings.Replace(filename, name+".", "", 1)
	return time.Parse(backupTimeFormat, ts)
}

// dir returns the directory for the current filename.
func (r *Roller) dir() string {
	return filepath.Dir(r.filename)
}

// logInfo is a convenience struct to return the filename and its embedded
// timestamp.
type logInfo struct {
	timestamp time.Time
	os.FileInfo
}

// byFormatTime sorts by newest time formatted in the name.
type byFormatTime []logInfo

func (b byFormatTime) Less(i, j int) bool {
	return b[i].timestamp.After(b[j].timestamp)
}

func (b byFormatTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byFormatTime) Len() int {
	return len(b)
}
