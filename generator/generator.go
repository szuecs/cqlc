package generator

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gocql/gocql"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrInvalidOptions = errors.New("invalid options")
)

type Options struct {
	Instance string `short:"i" long:"instance" description:"The Cassandra instance to connect to"`
	Keyspace string `short:"k" long:"keyspace" description:"The keyspace that contains the target schema"`
	Package  string `short:"p" long:"package" description:"The name of the target package for the generated code"`
	Output   string `short:"o" long:"output" description:"The file to write the generated bindings to"`
	Version  func() `short:"V" long:"version" description:"Print cqlc version and exit"`
	Verbose  []bool `short:"v" long:"verbose" description:"Show verbose debug information"`
	Symbols  bool   `short:"s" long:"symbols" description:"Generate compile symbols for each column family"`
	Username string `short:"u" long:"username" description:"Username for authentication"`
	Password string `short:"w" long:"password" description:"Password for authentication"`
}

type Provenance struct {
	Keyspace      string
	Version       string
	Timestamp     time.Time
	NegotiatedCQL string
	ServerCQL     string
	ServerRelease string
	HostId        gocql.UUID
}

func Generate(opts *Options, version string) error {

	err := validateOptions(opts)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	if err = generateBinding(opts, version, &b); err != nil {
		return err
	}
	if b.Len() > 0 {
		if err := os.MkdirAll(filepath.Dir(opts.Output), os.ModePerm); err != nil {
			return err
		}
		if err := ioutil.WriteFile(opts.Output, b.Bytes(), os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func validateOptions(opts *Options) error {
	if opts.Instance == "" || opts.Keyspace == "" || opts.Package == "" || opts.Output == "" {
		return ErrInvalidOptions
	}
	if (opts.Username == "" && opts.Password != "") || (opts.Username != "" && opts.Password == "") {
		return ErrInvalidOptions
	}
	return nil
}

func coalesceImports(md *gocql.KeyspaceMetadata) []string {

	set := make(map[string]bool)
	for _, path := range importPaths(md) {
		set[path] = true
	}

	set["github.com/relops/cqlc/cqlc"] = true
	set["github.com/gocql/gocql"] = true
	set["log"] = true

	paths := make([]string, 0)
	for path, _ := range set {
		paths = append(paths, path)
	}

	return paths
}

func generateBinding(opts *Options, version string, w io.Writer) error {

	cluster := gocql.NewCluster(opts.Instance)

	if opts.Username != "" && opts.Password != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: opts.Username,
			Password: opts.Password,
		}
	}

	// #############################
	// TODO Temporary hack until upstream API is fixed
	cluster.Keyspace = opts.Keyspace
	// #############################

	s, err := cluster.CreateSession()
	defer s.Close()

	if err != nil {
		return fmt.Errorf("Connect error", err)
	}

	var release, cqlVersion string
	var hostId gocql.UUID
	err = s.Query(`SELECT release_version, cql_version, host_id
		           FROM system.local`).Scan(&release, &cqlVersion, &hostId)
	if err != nil {
		return fmt.Errorf("System metadata error", err)
	}

	md, err := s.KeyspaceMetadata()

	if err != nil {
		return err
	}

	provenance := Provenance{
		Keyspace:      opts.Keyspace,
		Version:       version,
		Timestamp:     time.Now(),
		HostId:        hostId,
		NegotiatedCQL: cluster.CQLVersion,
		ServerCQL:     cqlVersion,
		ServerRelease: release,
	}

	meta := make(map[string]interface{})
	meta["Provenance"] = provenance
	meta["Options"] = opts
	meta["Imports"] = coalesceImports(md)
	meta["Tables"] = md.Tables

	var b bytes.Buffer
	if err := bindingTemplate.Execute(&b, meta); err != nil {
		return err
	}

	bfmt, err := format.Source(b.Bytes())
	if err != nil {
		return err
	}

	if _, err := w.Write(bfmt); err != nil {
		return err
	}

	return nil
}

func importPaths(md *gocql.KeyspaceMetadata) (imports []string) {
	// Ideally need to use a set
	paths := make(map[string]bool)

	f := func(t *gocql.TypeInfo) {
		literal := literalTypes[t.Type]
		if t.Type == gocql.TypeCustom && strings.Contains(t.Custom, "TimestampType") {
			fmt.Println("Cannot resolve timestamp until #314 has landed on #309")
		}
		if strings.Contains(literal, ".") {
			paths[literal] = true
		}
	}

	for _, t := range md.Tables {
		for _, col := range t.Columns {
			t := col.Type
			//fmt.Printf("Resolved col type: %s\n", t.Type)
			switch t.Type {
			case gocql.TypeList, gocql.TypeSet:
				f(t.Elem)
			case gocql.TypeMap:
				f(t.Key)
				f(t.Elem)
			default:
				f(&t)
			}
		}
	}

	for path, _ := range paths {
		customPath, present := customImportPaths[path]
		if present {
			imports = append(imports, customPath)
		} else {
			parts := strings.Split(path, ".")
			imports = append(imports, parts[0])
		}
	}

	return imports
}
