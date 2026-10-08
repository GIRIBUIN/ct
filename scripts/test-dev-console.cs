// Windows-only test harness. Give PowerShell a real, hidden console from
// process startup; redirected-stdin tests do not exercise the same code path.
using System;
using System.ComponentModel;
using System.Diagnostics;
using System.IO;
using System.Runtime.InteropServices;
using System.Text;

public static class CtDevConsole
{
    [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
    private struct StartupInfo
    {
        public int Size;
        public string Reserved, Desktop, Title;
        public uint X, Y, XSize, YSize, XCountChars, YCountChars, FillAttribute, Flags;
        public ushort ShowWindow, ReservedSize;
        public IntPtr ReservedPointer, Stdin, Stdout, Stderr;
    }

    [StructLayout(LayoutKind.Sequential)]
    private struct ProcessInfo
    {
        public IntPtr Process, Thread;
        public uint ProcessId, ThreadId;
    }

    [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
    private struct KeyEvent
    {
        [MarshalAs(UnmanagedType.Bool)] public bool Down;
        public ushort Repeat, VirtualKey, Scan;
        public char Character;
        public uint ControlState;
    }

    [StructLayout(LayoutKind.Explicit, Size = 20)]
    private struct InputRecord
    {
        [FieldOffset(0)] public ushort Type;
        [FieldOffset(4)] public KeyEvent Key;
    }

    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    [return: MarshalAs(UnmanagedType.Bool)]
    private static extern bool CreateProcessW(string application, StringBuilder command,
        IntPtr processAttributes, IntPtr threadAttributes, bool inheritHandles, uint flags,
        IntPtr environment, string directory, ref StartupInfo startup, out ProcessInfo process);

    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern uint WaitForSingleObject(IntPtr handle, uint milliseconds);
    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool GetExitCodeProcess(IntPtr process, out uint code);
    [DllImport("kernel32.dll")]
    private static extern bool CloseHandle(IntPtr handle);
    [DllImport("kernel32.dll")]
    private static extern IntPtr GetStdHandle(int id);
    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool GetConsoleMode(IntPtr handle, out uint mode);
    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern bool WriteConsoleInputW(IntPtr input, InputRecord[] records,
        uint count, out uint written);

    public static void QueueInput(string text)
    {
        IntPtr input = GetStdHandle(-10);
        uint mode;
        if (!GetConsoleMode(input, out mode))
            throw new Win32Exception(Marshal.GetLastWin32Error(), "Test stdin is not a real console");
        text = text.Replace("\r\n", "\n").Replace('\n', '\r');
        InputRecord[] records = new InputRecord[text.Length * 2];
        for (int i = 0; i < text.Length; i++)
        {
            KeyEvent key = new KeyEvent { Down = true, Repeat = 1, Character = text[i] };
            if (text[i] == '\r') key.VirtualKey = 13;
            records[i * 2] = new InputRecord { Type = 1, Key = key };
            key.Down = false;
            records[i * 2 + 1] = new InputRecord { Type = 1, Key = key };
        }
        uint written;
        if (!WriteConsoleInputW(input, records, (uint)records.Length, out written) || written != records.Length)
            throw new Win32Exception(Marshal.GetLastWin32Error(), "Cannot queue console keyboard input");
    }

    public static int RunHidden(string application, string arguments, string directory)
    {
        StartupInfo startup = new StartupInfo();
        startup.Size = Marshal.SizeOf(typeof(StartupInfo));
        startup.Flags = 1; // STARTF_USESHOWWINDOW, SW_HIDE = 0
        ProcessInfo process;
        if (!CreateProcessW(application, new StringBuilder("\"" + application + "\" " + arguments),
            IntPtr.Zero, IntPtr.Zero, false, 0x10 /* CREATE_NEW_CONSOLE */,
            IntPtr.Zero, directory, ref startup, out process))
            throw new Win32Exception(Marshal.GetLastWin32Error());
        try
        {
            if (WaitForSingleObject(process.Process, 30000) != 0)
            {
                // Terminate only the process tree created by this test.
                ProcessStartInfo kill = new ProcessStartInfo(
                    Path.Combine(Environment.SystemDirectory, "taskkill.exe"),
                    "/PID " + process.ProcessId + " /T /F");
                kill.UseShellExecute = false;
                kill.CreateNoWindow = true;
                using (Process cleanup = Process.Start(kill)) { cleanup.WaitForExit(); }
                throw new TimeoutException("Console development command timed out");
            }
            uint code;
            if (!GetExitCodeProcess(process.Process, out code))
                throw new Win32Exception(Marshal.GetLastWin32Error());
            return (int)code;
        }
        finally
        {
            CloseHandle(process.Thread);
            CloseHandle(process.Process);
        }
    }
}
