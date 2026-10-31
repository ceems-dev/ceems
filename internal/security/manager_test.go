package security

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/ceems-dev/ceems/internal/osexec"
	"github.com/steiler/acls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"kernel.org/pub/linux/libs/security/libcap/cap"
)

func skipUnprivileged(t *testing.T) {
	t.Helper()

	// Get current user
	currentUser, err := user.Current()
	require.NoError(t, err)

	if currentUser.Uid != "0" {
		t.Skip("Skipping testing due to lack of privileges")
	}
}

func testConfig(tmpDir string) (*Config, error) {
	// Make test directories
	testDir := filepath.Join(tmpDir, "l1", "l2", "l3")

	err := os.MkdirAll(testDir, 0o700)
	if err != nil {
		return nil, err
	}

	// Add rx on tmpDir/l1/l2
	err = os.Chmod(filepath.Join(tmpDir, "l1", "l2"), 0o705)
	if err != nil {
		return nil, err
	}

	// Create a file in testDir
	testReadFile := filepath.Join(testDir, "testRead")

	err = os.WriteFile(testReadFile, []byte("hello"), 0o700)
	if err != nil {
		return nil, err
	}

	testWriteFile := filepath.Join(testDir, "testWrite")

	err = os.WriteFile(testWriteFile, []byte("hello"), 0o700)
	if err != nil {
		return nil, err
	}

	return &Config{
		RunAsUser: "nobody",
		ReadPaths: []string{
			testReadFile,
			filepath.Join(tmpDir, "l1", "l2", "l3"),
			filepath.Join(tmpDir, "l1", "l2"),
			filepath.Join(tmpDir, "l1"),
			filepath.Dir(tmpDir),
			tmpDir,
		},
		ReadWritePaths: []string{
			testWriteFile,
		},
	}, nil
}

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	c, err := testConfig(tmpDir)
	require.NoError(t, err)

	m, err := NewManager(c, noOpLogger)
	require.NoError(t, err)

	expectedEntries := []acl{
		{path: filepath.Join(tmpDir, "l1", "l2", "l3"), entry: acls.NewEntry(acls.TAG_ACL_USER, 65534, 5)},
		{path: filepath.Join(tmpDir, "l1", "l2"), entry: acls.NewEntry(acls.TAG_ACL_USER, 65534, 5)},
		{path: filepath.Join(tmpDir, "l1"), entry: acls.NewEntry(acls.TAG_ACL_USER, 65534, 5)},
		{path: tmpDir, entry: acls.NewEntry(acls.TAG_ACL_USER, 65534, 5)},
		{path: filepath.Dir(tmpDir), entry: acls.NewEntry(acls.TAG_ACL_USER, 65534, 5)},
		{path: filepath.Join(tmpDir, "l1", "l2", "l3", "testRead"), entry: acls.NewEntry(acls.TAG_ACL_USER, 65534, 4)},
		{path: filepath.Join(tmpDir, "l1", "l2", "l3", "testWrite"), entry: acls.NewEntry(acls.TAG_ACL_USER, 65534, 6)},
	}

	assert.ElementsMatch(t, expectedEntries, m.acls)

	// Test illegal runAsUser
	c.RunAsUser = "illegal"

	_, err = NewManager(c, noOpLogger)
	require.Error(t, err)
}

func TestDefaultCaps(t *testing.T) {
	tmpDir := t.TempDir()

	// Expected in case of systemd >= 254 and in desktop env
	capSet, err := cap.FromText("cap_wake_alarm=i")
	require.NoError(t, err)

	// Empty capset
	emptyCapSet := cap.NewSet()

	tests := []struct {
		name       string
		cmdOut     string
		displayEnv bool
		expected   *cap.Set
	}{
		{
			name: "systemd > 254 in Desktop env",
			cmdOut: `systemd 259 (259.9-0ubuntu0.1)
+PAM +AUDIT +SELINUX +APPARMOR +IMA +IPE +SMACK +SECCOMP +GCRYPT -GNUTLS +OPENSSL +ACL +BLKID +CURL +ELFUTILS +FIDO2 +IDN2 -IDN +KMOD +LIBCRYPTSETUP +LIBCRYPTSETUP_PLUGINS +LIBFDISK +PCRE2 +PWQUALITY +P11KIT +QRENCODE +TPM2 +BZIP2 +LZ4 +XZ +ZLIB +ZSTD +BPF_FRAMEWORK +BTF -XKBCOMMON -UTMP +SYSVINIT +LIBARCHIVE`,
			displayEnv: true,
			expected:   capSet,
		},
		{
			name: "systemd = 254 in Desktop env",
			cmdOut: `systemd 254 (254.9-0ubuntu0.1)
+PAM +AUDIT +SELINUX +APPARMOR +IMA +IPE +SMACK +SECCOMP +GCRYPT -GNUTLS +OPENSSL +ACL +BLKID +CURL +ELFUTILS +FIDO2 +IDN2 -IDN +KMOD +LIBCRYPTSETUP +LIBCRYPTSETUP_PLUGINS +LIBFDISK +PCRE2 +PWQUALITY +P11KIT +QRENCODE +TPM2 +BZIP2 +LZ4 +XZ +ZLIB +ZSTD +BPF_FRAMEWORK +BTF -XKBCOMMON -UTMP +SYSVINIT +LIBARCHIVE`,
			displayEnv: true,
			expected:   capSet,
		},
		{
			name: "systemd < 254 in Desktop env",
			cmdOut: `systemd 252 (252.9-0ubuntu0.1)
		+PAM +AUDIT +SELINUX +APPARMOR +IMA +IPE +SMACK +SECCOMP +GCRYPT -GNUTLS +OPENSSL +ACL +BLKID +CURL +ELFUTILS +FIDO2 +IDN2 -IDN +KMOD +LIBCRYPTSETUP +LIBCRYPTSETUP_PLUGINS +LIBFDISK +PCRE2 +PWQUALITY +P11KIT +QRENCODE +TPM2 +BZIP2 +LZ4 +XZ +ZLIB +ZSTD +BPF_FRAMEWORK +BTF -XKBCOMMON -UTMP +SYSVINIT +LIBARCHIVE`,
			displayEnv: true,
			expected:   emptyCapSet,
		},
		{
			name: "systemd > 254 in headless env",
			cmdOut: `systemd 259 (259.9-0ubuntu0.1)
		+PAM +AUDIT +SELINUX +APPARMOR +IMA +IPE +SMACK +SECCOMP +GCRYPT -GNUTLS +OPENSSL +ACL +BLKID +CURL +ELFUTILS +FIDO2 +IDN2 -IDN +KMOD +LIBCRYPTSETUP +LIBCRYPTSETUP_PLUGINS +LIBFDISK +PCRE2 +PWQUALITY +P11KIT +QRENCODE +TPM2 +BZIP2 +LZ4 +XZ +ZLIB +ZSTD +BPF_FRAMEWORK +BTF -XKBCOMMON -UTMP +SYSVINIT +LIBARCHIVE`,
			displayEnv: false,
			expected:   emptyCapSet,
		},
		{
			name: "systemd = 254 in headless env",
			cmdOut: `systemd 254 (254.9-0ubuntu0.1)
		+PAM +AUDIT +SELINUX +APPARMOR +IMA +IPE +SMACK +SECCOMP +GCRYPT -GNUTLS +OPENSSL +ACL +BLKID +CURL +ELFUTILS +FIDO2 +IDN2 -IDN +KMOD +LIBCRYPTSETUP +LIBCRYPTSETUP_PLUGINS +LIBFDISK +PCRE2 +PWQUALITY +P11KIT +QRENCODE +TPM2 +BZIP2 +LZ4 +XZ +ZLIB +ZSTD +BPF_FRAMEWORK +BTF -XKBCOMMON -UTMP +SYSVINIT +LIBARCHIVE`,
			displayEnv: false,
			expected:   emptyCapSet,
		},
		{
			name: "systemd < 254 in headless env",
			cmdOut: `systemd 252 (252.9-0ubuntu0.1)
		+PAM +AUDIT +SELINUX +APPARMOR +IMA +IPE +SMACK +SECCOMP +GCRYPT -GNUTLS +OPENSSL +ACL +BLKID +CURL +ELFUTILS +FIDO2 +IDN2 -IDN +KMOD +LIBCRYPTSETUP +LIBCRYPTSETUP_PLUGINS +LIBFDISK +PCRE2 +PWQUALITY +P11KIT +QRENCODE +TPM2 +BZIP2 +LZ4 +XZ +ZLIB +ZSTD +BPF_FRAMEWORK +BTF -XKBCOMMON -UTMP +SYSVINIT +LIBARCHIVE`,
			displayEnv: false,
			expected:   emptyCapSet,
		},
		{
			name:       "no systemctl output",
			displayEnv: true,
			expected:   emptyCapSet,
		},
		{
			name: "illegal systemctl output",
			cmdOut: `systemd sdfsd (254.9-0ubuntu0.1)
		+PAM +AUDIT +SELINUX +APPARMOR +IMA +IPE +SMACK +SECCOMP +GCRYPT -GNUTLS +OPENSSL +ACL +BLKID +CURL +ELFUTILS +FIDO2 +IDN2 -IDN +KMOD +LIBCRYPTSETUP +LIBCRYPTSETUP_PLUGINS +LIBFDISK +PCRE2 +PWQUALITY +P11KIT +QRENCODE +TPM2 +BZIP2 +LZ4 +XZ +ZLIB +ZSTD +BPF_FRAMEWORK +BTF -XKBCOMMON -UTMP +SYSVINIT +LIBARCHIVE`,
			displayEnv: true,
			expected:   emptyCapSet,
		},
	}

	// Make a new test config
	c, err := testConfig(tmpDir)
	require.NoError(t, err)

	// Make a new manager
	m, err := NewManager(c, noOpLogger)
	require.NoError(t, err)

	// Override systemctl path
	m.systemctlBinPath = filepath.Join(tmpDir, "systemctl")

	for _, test := range tests {
		// Write content of systemctl --version to file
		content := fmt.Sprintf(`#!/bin/bash

echo """%s"""`, test.cmdOut)
		err = os.WriteFile(m.systemctlBinPath, []byte(content), 0o700)
		require.NoError(t, err, test.name)

		// Set/Unset display env var
		if test.displayEnv {
			t.Setenv("DISPLAY", ":0")
		} else {
			t.Setenv("DISPLAY", "")
		}

		// Get default caps
		got := m.defaultCaps()
		assert.Equal(t, test.expected, got, test.name)
	}
}

func TestACLs(t *testing.T) {
	skipUnprivileged(t)

	tmpDir := t.TempDir()

	c, err := testConfig(tmpDir)
	require.NoError(t, err)

	readFile := filepath.Join(tmpDir, "l1", "l2", "l3", "testRead")
	writeFile := filepath.Join(tmpDir, "l1", "l2", "l3", "testWrite")

	m, err := NewManager(c, noOpLogger)
	require.NoError(t, err)

	// Add ACL entries
	err = m.addACLEntries()
	require.NoError(t, err)

	// Now all paths should be reachable
	err = m.pathsReachable()
	require.NoError(t, err)

	// Attempt to read and write file
	_, err = osexec.ExecuteAs("cat", []string{readFile}, 65534, 65534, nil)
	require.NoError(t, err)

	_, err = osexec.ExecuteAs("touch", []string{writeFile}, 65534, 65534, nil)
	require.NoError(t, err)

	// Drop all ACLs
	err = m.DeleteACLEntries()
	require.NoError(t, err)

	// Now attempt to read and write files. Should be errors
	_, err = osexec.ExecuteAs("cat", []string{readFile}, 65534, 65534, nil)
	require.Error(t, err)

	_, err = osexec.ExecuteAs("touch", []string{writeFile}, 65534, 65534, nil)
	require.Error(t, err)
}

func TestGetDefaultRunAsUserPriv(t *testing.T) {
	skipUnprivileged(t)

	user, err := GetDefaultRunAsUser()
	require.NoError(t, err)

	assert.Equal(t, "nobody", user)
}

// func TestDropPrivileges(t *testing.T) {
// 	skipUnprivileged(t)

// 	// Target a cap
// 	value, err := cap.FromName("cap_sys_admin")
// 	require.NoError(t, err)

// 	// Make test config
// 	// We are running as root as using any other user
// 	// will make the process owner running that test
// 	// as that user and go wont be able to create
// 	// build and coverage related files anymore after
// 	// test finishes.
// 	cfg := Config{
// 		RunAsUser: "root",
// 		Caps:      []cap.Value{value},
// 	}

// 	// Drop all privileges
// 	err = DropPrivileges(&cfg)
// 	require.NoError(t, err)

// 	// Check process do not have any privileges
// 	capName := cap.GetProc().String()

// 	require.NoError(t, err)
// 	assert.EqualValues(t, "cap_sys_admin=p", capName)

// 	// Get current caps
// 	current := cap.GetProc()

// 	// Setback current caps
// 	err = current.SetFlag(cap.Effective, true, value)
// 	require.NoError(t, err)

// 	err = current.SetProc()
// 	require.NoError(t, err)
// }
