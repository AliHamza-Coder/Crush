; CRUSH NSIS Installer
; Installs crush.exe to Program Files and adds to PATH

!include "MUI2.nsh"

Name "CRUSH"
OutFile "crush-setup.exe"
InstallDir "$PROGRAMFILES64\CRUSH"
RequestExecutionLevel admin

!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "../../LICENSE.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section "Install"
    SetOutPath "$INSTDIR"
    
    ; Install files
    File "target\release\crush.exe"
    File "README.md"
    
    ; Create uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"
    
    ; Add to system PATH via registry
    ReadRegStr $0 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path"
    WriteRegStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path" "$0;$INSTDIR"
    
    ; Create Start Menu shortcuts
    CreateDirectory "$SMPROGRAMS\CRUSH"
    CreateShortcut "$SMPROGRAMS\CRUSH\CRUSH.lnk" "$INSTDIR\crush.exe"
    CreateShortcut "$SMPROGRAMS\CRUSH\Uninstall.lnk" "$INSTDIR\uninstall.exe"
    
    ; Registry for Add/Remove Programs
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CRUSH" "DisplayName" "CRUSH - Multimedia Mission Control"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CRUSH" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CRUSH" "InstallLocation" "$INSTDIR"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CRUSH" "DisplayVersion" "3.0.0"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CRUSH" "Publisher" "CRUSH"
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CRUSH" "NoModify" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CRUSH" "NoRepair" 1
SectionEnd

Section "Uninstall"
    ; Remove files
    Delete "$INSTDIR\crush.exe"
    Delete "$INSTDIR\README.md"
    Delete "$INSTDIR\uninstall.exe"
    RMDir "$INSTDIR"
    
    ; Remove from PATH via registry
    ReadRegStr $0 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path"
    StrReplace $0 "$0" ";$INSTDIR" ""
    StrReplace $0 "$0" "$INSTDIR;" ""
    StrReplace $0 "$0" "$INSTDIR" ""
    WriteRegStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path" "$0"
    
    ; Remove Start Menu shortcuts
    RMDir /r "$SMPROGRAMS\CRUSH"
    
    ; Remove registry entries
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CRUSH"
SectionEnd
