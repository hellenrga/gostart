// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ld

// This file provides helper functions for updating/rewriting the UUID
// load command within a Go go binary generated on Darwin using
// external linking. Why is it necessary to update the UUID load
// command? See issue #64947 for more detail, but the short answer is
// that newer versions of the Macos toolchain (the newer linker in
// particular) appear to compute the UUID based not just on the
// content of the object files being linked but also on things like
// the timestamps/paths of the objects; this makes it
// difficult/impossible to support reproducible builds. Since we try
// hard to maintain build reproducibility for Go, the APIs here
// compute a new UUID (based on the Go build ID) and write it to the
// final executable generated by the external linker.

import (
	"cmd/internal/notsha256"
	"debug/macho"
	"io"
	"os"
	"unsafe"
)

// uuidFromGoBuildId hashes the Go build ID and returns a slice of 16
// bytes suitable for use as the payload in a Macho LC_UUID load
// command.
func uuidFromGoBuildId(buildID string) []byte {
	if buildID == "" {
		return make([]byte, 16)
	}
	hashedBuildID := notsha256.Sum256([]byte(buildID))
	rv := hashedBuildID[:16]

	// RFC 4122 conformance (see RFC 4122 Sections 4.2.2, 4.1.3). We
	// want the "version" of this UUID to appear as 'hashed' as opposed
	// to random or time-based.  This is something of a fiction since
	// we're not actually hashing using MD5 or SHA1, but it seems better
	// to use this UUID flavor than any of the others. This is similar
	// to how other linkers handle this (for example this code in lld:
	// https://github.com/llvm/llvm-project/blob/2a3a79ce4c2149d7787d56f9841b66cacc9061d0/lld/MachO/Writer.cpp#L524).
	rv[6] &= 0x0f
	rv[6] |= 0x30
	rv[8] &= 0x3f
	rv[8] |= 0xc0

	return rv
}

// machoRewriteUuid copies over the contents of the Macho executable
// exef into the output file outexe, and in the process updates the
// LC_UUID command to a new value recomputed from the Go build id.
func machoRewriteUuid(ctxt *Link, exef *os.File, exem *macho.File, outexe string) error {
	outf, err := os.OpenFile(outexe, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer outf.Close()

	// Copy over the file.
	if _, err := io.Copy(outf, exef); err != nil {
		return err
	}

	// Locate the portion of the binary containing the load commands.
	cmdOffset := unsafe.Sizeof(exem.FileHeader)
	if is64bit := exem.Magic == macho.Magic64; is64bit {
		// mach_header_64 has one extra uint32.
		cmdOffset += unsafe.Sizeof(exem.Magic)
	}
	if _, err := outf.Seek(int64(cmdOffset), 0); err != nil {
		return err
	}

	// Read the load commands, looking for the LC_UUID cmd. If/when we
	// locate it, overwrite it with a new value produced by
	// uuidFromGoBuildId.
	reader := loadCmdReader{next: int64(cmdOffset),
		f: outf, order: exem.ByteOrder}
	for i := uint32(0); i < exem.Ncmd; i++ {
		cmd, err := reader.Next()
		if err != nil {
			return err
		}
		if cmd.Cmd == LC_UUID {
			var u uuidCmd
			if err := reader.ReadAt(0, &u); err != nil {
				return err
			}
			copy(u.Uuid[:], uuidFromGoBuildId(*flagBuildid))
			if err := reader.WriteAt(0, &u); err != nil {
				return err
			}
			break
		}
	}

	// We're done
	return nil
}
