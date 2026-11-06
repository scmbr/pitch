// English is the source of truth: ru.ts must have exactly the same keys.
export const en = {
  'app.title': 'Pitch',
  'nav.label': "Main navigation",
  'training.title': "Training",
  'training.preparation': "Preparation, seconds",
  'training.speech': "Speech, seconds",
  'training.preview': "Camera preview",
  'training.preparing': "Preparation",
  'training.speaking': "Speech",
  'training.recording': "Recording",
  'training.paused': "Paused",
  'training.start': "Start training",
  'training.pause': "Pause",
  'training.resume': "Resume",
  'training.skip': "Skip preparation",
  'training.finish': "Finish and save",
  'training.abort': "Discard recording",
  'training.saved': "Recording saved locally.",
  'training.savedFile': "Video: videos/{name}",
  'training.discarded': "Recording discarded.",
  'training.cameraError': "Cannot open the camera or microphone. Check permissions and that the devices are available, then try again.",
  'training.diskError': "Recording could not be saved. Check free disk space and folder permissions, then start a new attempt.",
  'training.durationError': "Preparation must be 0–3600 seconds; speech must be 1–86400 seconds.",

  'settings.title': 'Settings',
  'settings.loading': 'Loading settings…',

  'settings.dataDir.label': 'Data folder',
  'settings.dataDir.hint':
    'Recordings, thumbnails and the database are stored here. Avoid folders that sync to the cloud (OneDrive, Dropbox, iCloud Drive): files may be uploaded or damaged.',
  'settings.dataDir.choose': 'Choose…',
  'settings.dataDir.pickerTitle': 'Choose the data folder',
  'settings.dataDir.active': 'In use now: {dir}',
  'settings.dataDir.restart': 'The new folder will be used after you restart Pitch.',

  'settings.language.label': 'Language',
  'settings.language.auto': 'Same as the system',
  'settings.language.ru': 'Русский',
  'settings.language.en': 'English',

  'settings.save': 'Save',
  'settings.saved': 'Settings saved.',

  'error.dataDirEmpty': 'Enter a data folder.',
  'error.dataDirNotAbsolute': 'The data folder must be a full path, for example C:\\Users\\you\\Pitch.',
  'error.dataDirNotWritable':
    'Pitch cannot write to this folder. Choose another folder or check that the drive is connected.',
  'error.languageUnknown': 'This language is not supported.',
  'error.saveFailed': 'Settings could not be saved. Check that the settings folder is writable.',
  'error.loadFailed': 'Settings could not be loaded.',
  'warning.settingsCorrupt': 'The settings file is damaged, so default settings are shown. Saving will replace it.',
} as const;

export type MessageKey = keyof typeof en;
