package update

import (
	"fmt"

	dbus "github.com/coreos/go.dbus"
)

func systemdDaemonReload() error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to session bus: %s", err.Error())
	}

	object := conn.Object("org.freedesktop.systemd1", "/org/freedesktop/systemd1")
	call := object.Call("org.freedesktop.systemd1.Manager.Reload", 0)

	if call.Err != nil {
		return call.Err
	}

	return nil
}

func systemdEnableUnits(units []string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to session bus: %s", err.Error())
	}

	object := conn.Object("org.freedesktop.systemd1", "/org/freedesktop/systemd1")
	call := object.Call("org.freedesktop.systemd1.Manager.EnableUnitFiles", 0, units, false, true)

	if call.Err != nil {
		return fmt.Errorf("enabling units %v: %s", units, call.Err.Error())
	}

	return nil
}

func systemdGetUnitPath(unitName string) (dbus.ObjectPath, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return "", fmt.Errorf("Failed to connect to session bus: %s", err.Error())
	}

	object := conn.Object("org.freedesktop.systemd1", "/org/freedesktop/systemd1")
	call := object.Call("org.freedesktop.systemd1.Manager.GetUnit", 0, unitName)

	if call.Err != nil {
		return "", call.Err
	}

	if len(call.Body) == 0 {
		return "", fmt.Errorf("systemdGetUnitPath: dbus gave an empty response")
	}

	path, ok := call.Body[0].(dbus.ObjectPath)
	if !ok {
		return "", fmt.Errorf("systemdGetUnitPath: dbus returned a non-ObjectPath")
	}

	return path, nil
}

func systemdStopUnit(unitName string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("Failed to connect to session bus: %s", err.Error())
	}

	unitPath, err := systemdGetUnitPath(unitName)
	if err != nil {
		return err
	}

	object := conn.Object("org.freedesktop.systemd1", unitPath)
	call := object.Call("org.freedesktop.systemd1.Unit.Stop", 0, "replace")

	if call.Err != nil {
		return call.Err
	}

	return nil
}

func systemdRestartUnit(unitName string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("Failed to connect to session bus: %s", err.Error())
	}

	unitPath, err := systemdGetUnitPath(unitName)
	if err != nil {
		return err
	}

	object := conn.Object("org.freedesktop.systemd1", unitPath)
	call := object.Call("org.freedesktop.systemd1.Unit.Restart", 0, "replace")

	if call.Err != nil {
		return call.Err
	}

	return nil
}
