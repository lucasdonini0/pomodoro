import type { Alarm } from "./types";

let context: AudioContext | undefined;
let alarmBus: GainNode | undefined;
let alarmInterval = 0;

function ready() {
  context ??= new AudioContext();
  void context.resume();
  return context;
}

function note(
  frequency: number,
  start: number,
  duration: number,
  volume: number,
  output: AudioNode,
  wave: OscillatorType = "sine",
) {
  const audio = ready();
  const oscillator = audio.createOscillator();
  const gain = audio.createGain();
  oscillator.type = wave;
  oscillator.frequency.setValueAtTime(frequency, start);
  gain.gain.setValueAtTime(0.0001, start);
  gain.gain.exponentialRampToValueAtTime(Math.max(0.0001, volume), start + 0.025);
  gain.gain.exponentialRampToValueAtTime(0.0001, start + duration);
  oscillator.connect(gain).connect(output);
  oscillator.start(start);
  oscillator.stop(start + duration + 0.01);
}

export function unlockAudio() {
  ready();
}

export function clickSound(volume: number) {
  if (volume <= 0) return;
  const audio = ready();
  note(620, audio.currentTime, 0.065, volume * 0.055, audio.destination);
  note(920, audio.currentTime + 0.025, 0.075, volume * 0.025, audio.destination);
}

export function completedSound(volume: number) {
  if (volume <= 0) return;
  const audio = ready();
  for (let repeat = 0; repeat < 3; repeat++) {
    const start = audio.currentTime + repeat * 0.85;
    note(660, start, 0.46, volume * 0.17, audio.destination);
    note(880, start + 0.24, 0.57, volume * 0.18, audio.destination);
  }
}

const melodies: Record<string, number[]> = {
  suave: [523, 659, 784, 659],
  sinos: [784, 988, 1175, 988],
  aurora: [440, 554, 659, 880],
  digital: [698, 698, 932, 932],
};

export function stopAlarmSound() {
  window.clearInterval(alarmInterval);
  alarmInterval = 0;
  if (alarmBus && context) {
    alarmBus.gain.cancelScheduledValues(context.currentTime);
    alarmBus.gain.setValueAtTime(alarmBus.gain.value, context.currentTime);
    alarmBus.gain.linearRampToValueAtTime(0, context.currentTime + 0.04);
  }
  alarmBus = undefined;
}

export function startAlarmSound(alarm: Alarm) {
  stopAlarmSound();
  if (alarm.volume <= 0) return;
  const audio = ready();
  const bus = audio.createGain();
  bus.gain.value = 1;
  bus.connect(audio.destination);
  alarmBus = bus;
  const names = Object.keys(melodies);
  const sound = alarm.sound === "random"
    ? names[Math.floor(Math.random() * names.length)]
    : alarm.sound;
  const melody = melodies[sound] ?? melodies.suave;
  const play = () => {
    const start = audio.currentTime + 0.04;
    melody.forEach((frequency, index) => {
      const offset = index * 0.42;
      note(frequency, start + offset, 0.7, alarm.volume * 0.19, bus);
      if (sound === "sinos") note(frequency * 2, start + offset, 0.45, alarm.volume * 0.045, bus);
    });
  };
  play();
  if (alarm.repeat) alarmInterval = window.setInterval(play, 2600);
}
