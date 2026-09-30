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
  endFrequency?: number,
  sustain = false,
) {
  const audio = ready();
  const oscillator = audio.createOscillator();
  const gain = audio.createGain();
  oscillator.type = wave;
  oscillator.frequency.setValueAtTime(frequency, start);
  if (endFrequency) oscillator.frequency.linearRampToValueAtTime(endFrequency, start + duration * 0.85);
  gain.gain.setValueAtTime(0.0001, start);
  gain.gain.exponentialRampToValueAtTime(Math.max(0.0001, volume), start + 0.02);
  if (sustain) gain.gain.setValueAtTime(Math.max(0.0001, volume), start + duration * 0.7);
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

const alarmSounds = ["alerta", "sirene", "campainha"] as const;

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
  const sound = alarm.sound === "random"
    ? alarmSounds[Math.floor(Math.random() * alarmSounds.length)]
    : alarm.sound;
  const play = () => {
    const start = audio.currentTime + 0.04;
    if (sound === "sirene") {
      for (const offset of [0, 0.9]) {
        note(620, start + offset, 0.78, alarm.volume * 0.42, bus, "sawtooth", 1120, true);
        note(310, start + offset, 0.78, alarm.volume * 0.11, bus, "square", 560, true);
      }
    } else if (sound === "campainha") {
      for (const offset of [0, 0.35, 1.05, 1.4]) {
        note(880, start + offset, 0.48, alarm.volume * 0.4, bus, "sawtooth");
        note(1320, start + offset, 0.42, alarm.volume * 0.19, bus);
      }
    } else {
      [0, 0.23, 0.46, 1.02, 1.25, 1.48].forEach((offset, index) => {
        note(index % 3 === 2 ? 1040 : 880, start + offset, 0.19, alarm.volume * 0.48, bus, "square", undefined, true);
        note(440, start + offset, 0.19, alarm.volume * 0.12, bus, "sine", undefined, true);
      });
    }
  };
  play();
  if (alarm.repeat) alarmInterval = window.setInterval(play, 2500);
}
