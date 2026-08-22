; Inno Setup script for Atom Neo.
;
; Build it with:
;   go build -ldflags "-s -w -X Atom3/src/misc.Version=v3.0.0" -o installer\atom.exe .\src
;   "C:\Program Files\Inno Setup 7\ISCC.exe" /DAppVersion=3.0.0 installer\atom-neo.iss
;
; The version comes in from the command line so a tag and the installer cannot
; drift apart. Without it the build falls back to a dev version.

#ifndef AppVersion
  #define AppVersion "0.0.0-dev"
#endif

#define AppName "Atom Neo"
; The directory and the PATH entry stay unspaced, because a space in PATH
; breaks anything that splits it naively.
#define AppDir "AtomNeo"
; Major and minor only, the way Python labels its Start Menu entry.
#define LangVersion "3.0"
#define ArchLabel "64-bit"
#define ProgId "AtomNeo.Script"
#define AppPublisher "cfm-miku-en"
#define AppURL "https://github.com/cfm-miku-en/Atom-Neo"
#define AppExe "atom.exe"
#define ManagerExe "AtomManager.exe"

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
DefaultDirName={localappdata}\{#AppDir}
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
SetupIconFile=atom.ico
UninstallDisplayName={#AppName} {#AppVersion}
UninstallDisplayIcon={app}\bin\atom.ico

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "addtopath"; Description: "Add atom to my PATH so I can run it from any terminal"; GroupDescription: "Setup:"
Name: "associate"; Description: "Open .atom files with Atom Neo"; GroupDescription: "Setup:"
Name: "associate\verb"; Description: "Add a right click ""Run and keep the window open"""; GroupDescription: "Setup:"
Name: "associate\keepopen"; Description: "Keep the window open on a double click too"; GroupDescription: "Setup:"; Flags: unchecked
Name: "manager"; Description: "Install Atom Manager, for changing any of this later"; GroupDescription: "Setup:"

[Files]
Source: "atom.exe"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "atom.ico"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\docs\*"; DestDir: "{app}\docs"; Flags: ignoreversion recursesubdirs
Source: "..\packages\*"; DestDir: "{app}\packages"; Flags: ignoreversion recursesubdirs

; The manager is this program kept on disk, so the settings pages have one
; implementation rather than two that drift.
Source: "{srcexe}"; DestDir: "{app}"; DestName: "{#ManagerExe}"; \
    Flags: external ignoreversion; Tasks: manager; Check: NotAlreadyTheManager

[Registry]
; Written to the user environment rather than the machine one, and only when the
; entry is not already there, so repeated installs cannot pile up duplicates.
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; \
    ValueData: "{olddata};{app}\bin"; Tasks: addtopath; Check: NeedsAddPath(ExpandConstant('{app}\bin'))

; Listed whether or not the extension was claimed, so atom still appears under
; Open with for anyone who would rather keep their own association.
Root: HKCU; Subkey: "Software\Classes\Applications\{#AppExe}\shell\open\command"; ValueType: string; ValueName: ""; \
    ValueData: """{app}\bin\{#AppExe}"" ""%1"" %*"; Flags: uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\Applications\{#AppExe}\SupportedTypes"; ValueType: string; ValueName: ".atom"; \
    ValueData: ""; Flags: uninsdeletekey

[Icons]
Name: "{group}\Atom Terminal {#LangVersion} ({#ArchLabel})"; Filename: "{app}\bin\{#AppExe}"; Parameters: "repl"; IconFilename: "{app}\bin\atom.ico"
Name: "{group}\Atom Manager"; Filename: "{app}\{#ManagerExe}"; IconFilename: "{app}\bin\atom.ico"; Tasks: manager
Name: "{group}\Documentation"; Filename: "{app}\docs\reference.md"
Name: "{group}\Uninstall {#AppName}"; Filename: "{uninstallexe}"

[Run]
Filename: "{app}\bin\{#AppExe}"; Parameters: "repl"; Description: "Open Atom Terminal now"; \
    Flags: postinstall nowait skipifsilent unchecked

[Code]
const
  ProgId = '{#ProgId}';
  Classes = 'Software\Classes\';
  UninstallKey =
    'Software\Microsoft\Windows\CurrentVersion\Uninstall\' +
    '{9F2C4A61-7B3E-4D58-9A0C-2E6D4B1F8C33}_is1';
  ShellAssocChanged = $08000000;

var
  Installed: Boolean;

{ Explorer caches associations, so a new .atom icon would not appear until the
  next sign in without telling the shell to reread them. }
procedure SHChangeNotify(EventId: Integer; Flags: Cardinal; Item1, Item2: Cardinal);
  external 'SHChangeNotify@shell32.dll stdcall';

function BinDir: string;
begin
  Result := ExpandConstant('{app}\bin');
end;

function ExePath: string;
begin
  Result := BinDir + '\{#AppExe}';
end;

{ cmd keeps the console after the script exits. Without it a double click opens
  a window, runs, and closes before anything can be read, which is the oldest
  complaint about associating a scripting language. }
function OpenCommand(KeepOpen: Boolean): string;
begin
  if KeepOpen then
    Result := 'cmd.exe /k ""' + ExePath + '" "%1""'
  else
    Result := '"' + ExePath + '" "%1" %*';
end;

function AssociationOn: Boolean;
var
  Value: string;
begin
  Result := RegQueryStringValue(HKEY_CURRENT_USER, Classes + '.atom', '', Value) and
            (CompareText(Value, ProgId) = 0);
end;

function KeepOpenOn: Boolean;
var
  Value: string;
begin
  Result := RegQueryStringValue(HKEY_CURRENT_USER,
    Classes + ProgId + '\shell\open\command', '', Value) and
    (Pos('cmd.exe', Lowercase(Value)) = 1);
end;

function VerbOn: Boolean;
var
  Value: string;
begin
  Result := RegQueryStringValue(HKEY_CURRENT_USER,
    Classes + ProgId + '\shell\runkept\command', '', Value);
end;

function PathHasBin: Boolean;
var
  Existing: string;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Existing) then
  begin
    Result := False;
    exit;
  end;
  Result := Pos(';' + Lowercase(BinDir) + ';', ';' + Lowercase(Existing) + ';') > 0;
end;

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

function NotAlreadyTheManager: Boolean;
begin
  Result := CompareText(ExpandConstant('{srcexe}'),
                        ExpandConstant('{app}\{#ManagerExe}')) <> 0;
end;

procedure DropFromPath;
var
  Existing: string;
  Target: string;
  Position: Integer;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Existing) then
    exit;

  Target := BinDir;
  Position := Pos(';' + Lowercase(Target), ';' + Lowercase(Existing));
  if Position = 0 then
    exit;

  { Only this entry is taken out; the rest of PATH is written back untouched. }
  Delete(Existing, Position, Length(Target) + 1);
  RegWriteExpandStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Existing);
end;

procedure ClearAssociation;
begin
  RegDeleteKeyIncludingSubkeys(HKEY_CURRENT_USER, Classes + ProgId);
  RegDeleteValue(HKEY_CURRENT_USER, Classes + '.atom', '');
  RegDeleteValue(HKEY_CURRENT_USER, Classes + '.atom\OpenWithProgids', ProgId);
end;

procedure ApplyAssociation(Want, KeepOpen, Verb: Boolean);
begin
  if not Want then
  begin
    ClearAssociation;
    exit;
  end;

  RegWriteStringValue(HKEY_CURRENT_USER, Classes + '.atom', '', ProgId);
  RegWriteStringValue(HKEY_CURRENT_USER, Classes + '.atom\OpenWithProgids', ProgId, '');
  RegWriteStringValue(HKEY_CURRENT_USER, Classes + ProgId, '', 'Atom script');
  RegWriteStringValue(HKEY_CURRENT_USER, Classes + ProgId + '\DefaultIcon', '',
    BinDir + '\atom.ico');
  RegWriteStringValue(HKEY_CURRENT_USER, Classes + ProgId + '\shell\open\command', '',
    OpenCommand(KeepOpen));

  if Verb then
  begin
    RegWriteStringValue(HKEY_CURRENT_USER, Classes + ProgId + '\shell\runkept',
      'MUIVerb', 'Run and keep the window open');
    RegWriteStringValue(HKEY_CURRENT_USER, Classes + ProgId + '\shell\runkept\command', '',
      OpenCommand(True));
  end
  else
    RegDeleteKeyIncludingSubkeys(HKEY_CURRENT_USER, Classes + ProgId + '\shell\runkept');
end;

function InitializeSetup: Boolean;
var
  Where: string;
begin
  Installed := RegQueryStringValue(HKEY_CURRENT_USER, UninstallKey, 'InstallLocation', Where);
  Result := True;
end;

{ Re-running while installed lands on the same task list, with every box already
  reflecting what is currently set, so it reads as a settings page rather than a
  second install. }
procedure InitializeWizard;
var
  Selected: string;
begin
  if not Installed then
    exit;

  WizardForm.Caption := 'Atom Manager';
  WizardForm.SelectTasksLabel.Caption :=
    'Atom Neo is already installed. Change anything you like and press Next.';

  Selected := '';
  if PathHasBin then
    Selected := Selected + 'addtopath,';
  if AssociationOn then
  begin
    Selected := Selected + 'associate,';
    if VerbOn then
      Selected := Selected + 'associate\verb,';
    if KeepOpenOn then
      Selected := Selected + 'associate\keepopen,';
  end;
  if FileExists(ExpandConstant('{app}\{#ManagerExe}')) then
    Selected := Selected + 'manager,';

  WizardSelectTasks(Selected);
end;

function ShouldSkipPage(PageID: Integer): Boolean;
begin
  Result := Installed and ((PageID = wpLicense) or (PageID = wpSelectDir));
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep <> ssPostInstall then
    exit;

  ApplyAssociation(
    WizardIsTaskSelected('associate'),
    WizardIsTaskSelected('associate\keepopen'),
    WizardIsTaskSelected('associate\verb'));

  { Clearing the box has to undo the entry, or the manager could only ever turn
    things on. }
  if not WizardIsTaskSelected('addtopath') then
    DropFromPath;

  if not WizardIsTaskSelected('manager') then
    DeleteFile(ExpandConstant('{app}\{#ManagerExe}'));

  SHChangeNotify(ShellAssocChanged, 0, 0, 0);
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep <> usPostUninstall then
    exit;

  DropFromPath;
  ClearAssociation;
  DeleteFile(ExpandConstant('{app}\{#ManagerExe}'));
  SHChangeNotify(ShellAssocChanged, 0, 0, 0);
end;
