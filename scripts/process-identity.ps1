# Read-only identity for matching ETW records without confusing reused PIDs.
# https://learn.microsoft.com/windows/win32/api/winternl/nf-winternl-ntqueryinformationprocess
if (-not ('PitchProcessIdentity' -as [type])) {
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class PitchProcessIdentity {
 [DllImport("kernel32.dll",SetLastError=true)] static extern IntPtr OpenProcess(uint access,bool inherit,int pid);
 [DllImport("kernel32.dll")] static extern bool CloseHandle(IntPtr handle);
 [DllImport("ntdll.dll")] static extern int NtQueryInformationProcess(IntPtr process,int kind,IntPtr buffer,int length,out int needed);
 public static string Read(int pid,DateTime expectedCreation) {
  IntPtr handle=OpenProcess(0x1000,false,pid); if(handle==IntPtr.Zero)return null;
  IntPtr buffer=Marshal.AllocHGlobal(65536);
  try {
   int needed; if(NtQueryInformationProcess(handle,64,buffer,65536,out needed)!=0 || needed<24)return null;
   if(Marshal.ReadInt32(buffer,4)!=pid)return null;
   var created=DateTime.FromFileTimeUtc(Marshal.ReadInt64(buffer,16));
   if(Math.Abs((created-expectedCreation.ToUniversalTime()).TotalMilliseconds)>1)return null;
   return unchecked((ulong)Marshal.ReadInt64(buffer,8)).ToString(System.Globalization.CultureInfo.InvariantCulture);
  } finally {Marshal.FreeHGlobal(buffer);CloseHandle(handle);}
 }
}
'@
}
