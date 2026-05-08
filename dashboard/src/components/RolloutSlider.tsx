"use client";

interface RolloutSliderProps {
  value: number;
  onChange: (value: number) => void;
  disabled?: boolean;
}

export function RolloutSlider({
  value,
  onChange,
  disabled = false,
}: RolloutSliderProps) {
  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    onChange(Number(e.target.value));
  }

  const color =
    value === 0
      ? "text-zinc-500"
      : value < 25
      ? "text-blue-400"
      : value < 50
      ? "text-blue-400"
      : value < 75
      ? "text-yellow-400"
      : value < 100
      ? "text-orange-400"
      : "text-emerald-400";

  return (
    <div className={`space-y-2 ${disabled ? "opacity-50 pointer-events-none" : ""}`}>
      <div className="flex items-center justify-between">
        <label className="text-sm font-medium text-zinc-300">
          Rollout Percentage
        </label>
        <span className={`text-2xl font-bold tabular-nums ${color}`}>
          {value}%
        </span>
      </div>
      <input
        type="range"
        min={0}
        max={100}
        step={1}
        value={value}
        onChange={handleChange}
        className="w-full cursor-pointer"
        disabled={disabled}
      />
      <div className="flex justify-between text-xs text-zinc-600">
        <span>0%</span>
        <span>25%</span>
        <span>50%</span>
        <span>75%</span>
        <span>100%</span>
      </div>
    </div>
  );
}

export default RolloutSlider;
