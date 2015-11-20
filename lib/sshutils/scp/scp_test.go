/*
Copyright 2015 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package scp

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/gravitational/teleport/Godeps/_workspace/src/github.com/gravitational/log"
	. "github.com/gravitational/teleport/Godeps/_workspace/src/gopkg.in/check.v1"
)

func TestSCP(t *testing.T) { TestingT(t) }

type SCPSuite struct {
}

var _ = Suite(&SCPSuite{})

func (s *SCPSuite) SetUpSuite(c *C) {
	log.Initialize("console", "WARN")
}

func (s *SCPSuite) TestSendFile(c *C) {
	dir := c.MkDir()
	target := filepath.Join(dir, "target")

	contents := []byte("hello, send file!")

	err := ioutil.WriteFile(target, contents, 0666)
	c.Assert(err, IsNil)

	srv, err := New(Command{Source: true, Target: target})
	c.Assert(err, IsNil)

	outDir := c.MkDir()
	cmd, in, out, epipe := command("scp", "-v", "-t", outDir)

	errC := make(chan error, 2)
	rw := &combo{out, in}
	go func() {
		errC <- cmd.Run()
		log.Infof("run completed")
	}()

	go func() {
		for {
			io.Copy(log.GetLogger().Writer(log.SeverityInfo), epipe)
		}
	}()

	go func() {
		errC <- srv.Serve(rw)
		log.Infof("serve completed")
		in.Close()
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-time.After(time.Second):
			panic("timeout")
		case err := <-errC:
			c.Assert(err, IsNil)
		}
	}

	bytes, err := ioutil.ReadFile(filepath.Join(outDir, "target"))
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, string(contents))
}

func (s *SCPSuite) TestReceiveFile(c *C) {
	dir := c.MkDir()
	source := filepath.Join(dir, "target")

	contents := []byte("hello, file contents!")
	err := ioutil.WriteFile(source, contents, 0666)
	c.Assert(err, IsNil)

	outDir := c.MkDir() + "/"

	srv, err := New(Command{Sink: true, Target: outDir})
	c.Assert(err, IsNil)

	cmd, in, out, epipe := command("scp", "-v", "-f", source)

	errC := make(chan error, 2)
	rw := &combo{out, in}
	go func() {
		errC <- cmd.Run()
		log.Infof("run completed")
	}()

	go func() {
		for {
			io.Copy(log.GetLogger().Writer(log.SeverityInfo), epipe)
		}
	}()

	go func() {
		errC <- srv.Serve(rw)
		log.Infof("serve completed")
		in.Close()
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-time.After(time.Second):
			panic("timeout")
		case err := <-errC:
			c.Assert(err, IsNil)
		}
	}

	bytes, err := ioutil.ReadFile(filepath.Join(outDir, "target"))
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, string(contents))
}

func (s *SCPSuite) TestSendDir(c *C) {
	dir := c.MkDir()

	c.Assert(os.Mkdir(filepath.Join(dir, "target_dir"), 0777), IsNil)

	err := ioutil.WriteFile(
		filepath.Join(dir, "target_dir", "target1"), []byte("file 1"), 0666)
	c.Assert(err, IsNil)

	err = ioutil.WriteFile(
		filepath.Join(dir, "target2"), []byte("file 2"), 0666)
	c.Assert(err, IsNil)

	srv, err := New(Command{Source: true, Target: dir, Recursive: true})
	c.Assert(err, IsNil)

	outDir := c.MkDir()

	cmd, in, out, epipe := command("scp", "-v", "-r", "-t", outDir)

	errC := make(chan error, 2)
	rw := &combo{out, in}
	go func() {
		errC <- cmd.Run()
		log.Infof("run completed")
	}()

	go func() {
		for {
			io.Copy(log.GetLogger().Writer(log.SeverityInfo), epipe)
		}
	}()

	go func() {
		errC <- srv.Serve(rw)
		log.Infof("serve completed")
		in.Close()
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-time.After(time.Second):
			panic("timeout")
		case err := <-errC:
			c.Assert(err, IsNil)
		}
	}

	name := filepath.Base(dir)
	bytes, err := ioutil.ReadFile(filepath.Join(outDir, name, "target_dir", "target1"))
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, string("file 1"))

	bytes, err = ioutil.ReadFile(filepath.Join(outDir, name, "target2"))
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, string("file 2"))
}

func (s *SCPSuite) TestReceiveDir(c *C) {
	dir := c.MkDir()

	c.Assert(os.Mkdir(filepath.Join(dir, "target_dir"), 0777), IsNil)

	err := ioutil.WriteFile(
		filepath.Join(dir, "target_dir", "target1"), []byte("file 1"), 0666)
	c.Assert(err, IsNil)

	err = ioutil.WriteFile(
		filepath.Join(dir, "target2"), []byte("file 2"), 0666)
	c.Assert(err, IsNil)

	outDir := c.MkDir() + "/"

	srv, err := New(Command{Sink: true, Target: outDir, Recursive: true})
	c.Assert(err, IsNil)

	cmd, in, out, epipe := command("scp", "-v", "-r", "-f", dir)

	errC := make(chan error, 2)
	rw := &combo{out, in}
	go func() {
		errC <- cmd.Run()
		log.Infof("run completed")
	}()

	go func() {
		for {
			io.Copy(log.GetLogger().Writer(log.SeverityInfo), epipe)
		}
	}()

	go func() {
		errC <- srv.Serve(rw)
		log.Infof("serve completed")
		in.Close()
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-time.After(time.Second):
			panic("timeout")
		case err := <-errC:
			c.Assert(err, IsNil)
		}
	}

	name := filepath.Base(dir)
	bytes, err := ioutil.ReadFile(filepath.Join(outDir, name, "target_dir", "target1"))
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, string("file 1"))

	bytes, err = ioutil.ReadFile(filepath.Join(outDir, name, "target2"))
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, string("file 2"))
}

type combo struct {
	r io.Reader
	w io.Writer
}

func (c *combo) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

func (c *combo) Write(b []byte) (int, error) {
	return c.w.Write(b)
}

func command(name string, args ...string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, io.ReadCloser) {
	cmd := exec.Command(name, args...)

	in, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	out, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	epipe, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}

	return cmd, in, out, epipe
}