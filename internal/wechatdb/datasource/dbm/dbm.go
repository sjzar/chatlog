package dbm

import (
	"database/sql"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/pkg/filecopy"
	"github.com/sjzar/chatlog/pkg/filemonitor"
)

type DBManager struct {
	path    string
	fm      *filemonitor.FileMonitor
	fgs     map[string]*filemonitor.FileGroup
	dbs     map[string]*sql.DB
	dbPaths map[string][]string
	mutex   sync.RWMutex
}

func NewDBManager(path string) *DBManager {
	return &DBManager{
		path:    path,
		fm:      filemonitor.NewFileMonitor(),
		fgs:     make(map[string]*filemonitor.FileGroup),
		dbs:     make(map[string]*sql.DB),
		dbPaths: make(map[string][]string),
	}
}

func (d *DBManager) AddGroup(g *Group) error {
	fg, err := filemonitor.NewFileGroup(g.Name, d.path, g.Pattern, g.BlackList)
	if err != nil {
		return err
	}
	fg.AddCallback(d.Callback)
	d.fm.AddGroup(fg)
	d.mutex.Lock()
	d.fgs[g.Name] = fg
	d.mutex.Unlock()
	return nil
}

func (d *DBManager) AddCallback(name string, callback func(event fsnotify.Event) error) error {
	d.mutex.RLock()
	fg, ok := d.fgs[name]
	d.mutex.RUnlock()
	if !ok {
		return errors.FileGroupNotFound(name)
	}
	fg.AddCallback(callback)
	return nil
}

// GetDBs 根据组名获取多个数据库连接
func (d *DBManager) GetDBs(name string) ([]*sql.DB, error) {
	log.Debug().Str("group_name", name).Msg("开始获取数据库连接组")

	d.mutex.RLock()
	group, exists := d.fgs[name]
	d.mutex.RUnlock()

	if !exists {
		log.Error().Str("group_name", name).Msg("数据库组不存在")
		return nil, fmt.Errorf("group %s not found", name)
	}

	files, err := group.List()
	if err != nil {
		log.Error().Err(err).Str("group_name", name).Msg("获取数据库文件列表失败")
		return nil, err
	}

	log.Debug().Str("group_name", name).Int("file_count", len(files)).Msg("找到数据库组文件")

	var dbs []*sql.DB
	for i, file := range files {
		log.Debug().Str("group_name", name).Int("file_index", i).Str("file_path", file).Msg("尝试连接数据库文件")

		db, err := d.OpenDB(file)
		if err != nil {
			log.Error().Err(err).Str("group_name", name).Int("file_index", i).Str("file_path", file).Msg("连接数据库文件失败")
			continue
		}

		log.Debug().Str("group_name", name).Int("file_index", i).Str("file_path", file).Msg("成功连接数据库文件")
		dbs = append(dbs, db)
	}

	if len(dbs) == 0 {
		log.Error().Str("group_name", name).Int("total_files", len(files)).Msg("所有数据库文件连接都失败")
		return nil, fmt.Errorf("no db files available for group %s", name)
	}

	log.Info().Str("group_name", name).Int("connected_db_count", len(dbs)).Int("total_files", len(files)).Msg("成功获取数据库连接组")
	return dbs, nil
}

// GetDB 根据组名获取单个数据库连接
func (d *DBManager) GetDB(name string) (*sql.DB, error) {
	log.Debug().Str("group_name", name).Msg("开始获取单个数据库连接")

	dbs, err := d.GetDBs(name)
	if err != nil {
		log.Error().Err(err).Str("group_name", name).Msg("获取数据库连接组失败")
		return nil, err
	}

	if len(dbs) == 0 {
		log.Error().Str("group_name", name).Msg("数据库连接组为空")
		return nil, fmt.Errorf("no db available for group %s", name)
	}

	log.Debug().Str("group_name", name).Msg("返回第一个数据库连接")
	return dbs[0], nil
}

func (d *DBManager) GetDBPath(name string) ([]string, error) {
	d.mutex.RLock()
	dbPaths, ok := d.dbPaths[name]
	d.mutex.RUnlock()
	if !ok {
		d.mutex.RLock()
		fg, ok := d.fgs[name]
		d.mutex.RUnlock()
		if !ok {
			return nil, errors.FileGroupNotFound(name)
		}
		list, err := fg.List()
		if err != nil {
			return nil, errors.DBFileNotFound(d.path, fg.PatternStr, err)
		}
		if len(list) == 0 {
			return nil, errors.DBFileNotFound(d.path, fg.PatternStr, nil)
		}
		dbPaths = list
		d.mutex.Lock()
		d.dbPaths[name] = dbPaths
		d.mutex.Unlock()
	}
	return dbPaths, nil
}

func (d *DBManager) OpenDB(path string) (*sql.DB, error) {
	d.mutex.RLock()
	db, ok := d.dbs[path]
	d.mutex.RUnlock()
	if ok {
		return db, nil
	}
	var err error
	tempPath := path
	if runtime.GOOS == "windows" {
		tempPath, err = filecopy.GetTempCopy("chatlog", path)
		if err != nil {
			log.Err(err).Msgf("获取临时拷贝文件 %s 失败", path)
			return nil, err
		}
	}
	db, err = sql.Open("sqlite3", tempPath)
	if err != nil {
		log.Err(err).Msgf("连接数据库 %s 失败", path)
		return nil, err
	}
	d.mutex.Lock()
	d.dbs[path] = db
	d.mutex.Unlock()
	return db, nil
}

func (d *DBManager) Callback(event fsnotify.Event) error {
	if !event.Op.Has(fsnotify.Create) {
		return nil
	}

	d.mutex.Lock()
	db, ok := d.dbs[event.Name]
	if ok {
		delete(d.dbs, event.Name)
		go func(db *sql.DB) {
			time.Sleep(time.Second * 5)
			db.Close()
		}(db)
	}
	d.mutex.Unlock()

	return nil
}

func (d *DBManager) Start() error {
	return d.fm.Start()
}

func (d *DBManager) Stop() error {
	return d.fm.Stop()
}

func (d *DBManager) Close() error {
	for _, db := range d.dbs {
		db.Close()
	}
	return d.fm.Stop()
}
