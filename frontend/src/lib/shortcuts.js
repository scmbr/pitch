/** @param {{code:string,ctrlKey:boolean,altKey:boolean,shiftKey:boolean,metaKey:boolean}} event */
export function shortcut(event) {
  if (!/^(Key[A-Z]|Digit[0-9]|Space|Enter|Escape|Arrow(Left|Right|Up|Down)|F([1-9]|1[0-2]))$/.test(event.code)) return '';
  return [event.ctrlKey?'Ctrl':'',event.altKey?'Alt':'',event.shiftKey?'Shift':'',event.metaKey?'Meta':'',event.code].filter(Boolean).join('+');
}
