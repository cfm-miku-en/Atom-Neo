; Inno Setup script for Atom-Neo.
;
; Build it with:
;   go build -ldflags "-s -w -X Atom3/src/misc.Version=v0.1.0" -o installer\atom.exe .\src
;   "C:\Program Files\Inno Setup 7\ISCC.exe" /DAppVersion=0.1.0 installer\atom-neo.iss
;
; The version comes in from the command line so a tag and the installer cannot
; drift apart. Without it the build falls back to a dev version.

#ifndef AppVersion
  #define AppVersion "0.0.0-dev"
#endif

#define AppName "Atom-Neo"
#define AppPublisher "cfm-miku-en"
#define AppURL "https://github.com/cfm-miku-en/Atom-Neo"
#define AppExe "atom.exe"

[Setup]
AppId={{9F2C4A61-7B3E-4D58-9A0C-2E6D4B1F8C33}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL={#AppURL}
AppSupportURL={#AppURL}/issues
AppUpdatesURL={#AppURL}/releases

; Installs for one user into their profile, so there is no administrator prompt
; and no mode dialog to dismiss before the wizard appears. Offering the choice
; only leads somewhere that needs elevation we do not want.
PrivilegesRequired=lowest
DefaultDirName={localappdata}\{#AppName}
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
DisableDirPage=auto

OutputDir=..\dist
OutputBaseFilename=AtomNeoSetup-{#AppVersion}
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
ArchitecturesInstallIn64BitMode=x64compatible

LicenseFile=..\LICENSE
UninstallDisplayName={#AppName} {#AppVersion}
UninstallDisplayIcon={app}\bin\{#AppExe}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "addtopath"; Description: "Add atom to my PATH so I can run it from any terminal"; GroupDescription: "Setup:"

[Files]
Source: "atom.exe"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\docs\*"; DestDir: "{app}\docs"; Flags: ignoreversion recursesubdirs
Source: "..\packages\*"; DestDir: "{app}\packages"; Flags: ignoreversion recursesubdirs

[Registry]
; Written to the user environment rather than the machine one, and only when the
; entry is not already there, so repeated installs cannot pile up duplicates.
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; \
    ValueData: "{olddata};{app}\bin"; Tasks: addtopath; Check: NeedsAddPath(ExpandConstant('{app}\bin'))

[Icons]
Name: "{group}\{#AppName} repl"; Filename: "{app}\bin\{#AppExe}"; Parameters: "repl"
Name: "{group}\Documentation"; Filename: "{app}\docs\reference.md"
Name: "{group}\Uninstall {#AppName}"; Filename: "{uninstallexe}"

[Run]
Filename: "{app}\bin\{#AppExe}"; Parameters: "repl"; Description: "Try the repl now"; \
    Flags: postinstall nowait skipifsilent unchecked

[Code]
function NeedsAddPath(Param: string): Boolean;
var
  Existing: string;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Existing) then
  begin
    Result := True;
    exit;
  end;

  { Padded with semicolons on both sides so a folder is not mistaken for one
    whose name merely ends with the same text. }
  Result := Pos(';' + Lowercase(Param) + ';', ';' + Lowercase(Existing) + ';') = 0;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Existing: string;
  Target: string;
  Position: Integer;
begin
  if CurUninstallStep <> usPostUninstall then
    exit;

  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Existing) then
    exit;

  Target := ExpandConstant('{app}\bin');
  Position := Pos(';' + Lowercase(Target), ';' + Lowercase(Existing));
  if Position = 0 then
    exit;

  { Only this entry is taken out; the rest of PATH is written back untouched. }
  Delete(Existing, Position, Length(Target) + 1);
  RegWriteExpandStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Existing);
end;
