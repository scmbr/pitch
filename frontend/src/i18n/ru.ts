import type { MessageKey } from './en';

export const ru: Record<MessageKey, string> = {
  'app.title': 'Pitch',
  'nav.label': "Основная навигация",
  'training.title': "Тренировка",
  'training.preparation': "Подготовка, секунды",
  'training.speech': "Выступление, секунды",
  'training.preview': "Превью камеры",
  'training.preparing': "Подготовка",
  'training.speaking': "Выступление",
  'training.recording': "Идёт запись",
  'training.paused': "Пауза",
  'training.start': "Начать тренировку",
  'training.pause': "Пауза",
  'training.resume': "Продолжить",
  'training.skip': "Пропустить подготовку",
  'training.finish': "Завершить и сохранить",
  'training.abort': "Прервать без сохранения",
  'training.saved': "Запись сохранена на компьютере.",
  'training.savedFile': "Видео: videos/{name}",
  'training.discarded': "Запись удалена.",
  'training.cameraError': "Не удалось включить камеру или микрофон. Проверьте разрешения и доступность устройств и попробуйте снова.",
  'training.diskError': "Не удалось сохранить запись. Проверьте свободное место и доступ к папке и начните новую попытку.",
  'training.durationError': "Подготовка: от 0 до 3600 секунд. Выступление: от 1 до 86400 секунд.",

  'settings.title': 'Настройки',
  'settings.loading': 'Загружаю настройки…',

  'settings.dataDir.label': 'Папка данных',
  'settings.dataDir.hint':
    'Здесь хранятся записи, превью и база данных. Не выбирайте папки, которые синхронизируются с облаком (OneDrive, Dropbox, iCloud Drive): файлы могут уйти в облако или повредиться.',
  'settings.dataDir.choose': 'Выбрать…',
  'settings.dataDir.pickerTitle': 'Выберите папку данных',
  'settings.dataDir.active': 'Сейчас используется: {dir}',
  'settings.dataDir.restart': 'Новая папка будет использоваться после перезапуска Pitch.',

  'settings.language.label': 'Язык',
  'settings.language.auto': 'Как в системе',
  'settings.language.ru': 'Русский',
  'settings.language.en': 'English',

  'settings.save': 'Сохранить',
  'settings.saved': 'Настройки сохранены.',

  'error.dataDirEmpty': 'Укажите папку данных.',
  'error.dataDirNotAbsolute': 'Папка данных должна быть полным путём, например C:\\Users\\you\\Pitch.',
  'error.dataDirNotWritable':
    'Pitch не может записывать в эту папку. Выберите другую папку или проверьте, что диск подключён.',
  'error.languageUnknown': 'Этот язык не поддерживается.',
  'error.saveFailed': 'Не удалось сохранить настройки. Проверьте, что в папку настроек можно записывать.',
  'error.loadFailed': 'Не удалось загрузить настройки.',
  'warning.settingsCorrupt': 'Файл настроек повреждён, поэтому показаны настройки по умолчанию. Сохранение заменит его.',
};
