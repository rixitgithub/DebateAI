import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { X, Image as ImageIcon } from 'lucide-react';

interface AvatarModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectAvatar: (avatarUrl: string) => void;
  currentAvatar?: string;
}

const AvatarModal: React.FC<AvatarModalProps> = ({
  isOpen,
  onClose,
  onSelectAvatar,
  currentAvatar,
}) => {
  const [seed, setSeed] = useState(
    currentAvatar?.split('seed=')[1]?.split('&')[0] || 'Felix'
  );
  const [backgroundColor, setBackgroundColor] = useState('b6e3f4');
  const [nose, setNose] = useState('variant01');
  const [rotate, setRotate] = useState(0);
  const [scale, setScale] = useState(100);
  const [radius, setRadius] = useState(0);
  const [backgroundType, setBackgroundType] = useState('solid');
  const [selectedAvatar, setSelectedAvatar] = useState(
    currentAvatar || `https://api.dicebear.com/9.x/big-ears/svg?seed=${seed}`
  );

  const seeds = [
    'Felix',
    'Aneka',
    'Leah',
    'Jude',
    'Sadie',
    'Nolan',
    'Luis',
    'Robert',
    'Easton',
    'Eden',
    'Jocelyn',
    'Ryan',
    'Riley',
    'Chase',
    'George',
    'Kimberly',
    'Liam',
    'Avery',
    'Maria',
    'Eliza',
    'Brooklynn',
    'Vivian',
  ];

  const backgroundColors = ['b6e3f4', 'c0aede', 'd1d4f9', 'ffd5dc', 'ffdfbf'];
  const noseOptions = [
    'variant01',
    'variant02',
    'variant03',
    'variant04',
    'variant05',
    'variant06',
    'variant07',
    'variant08',
    'variant09',
    'variant10',
    'variant11',
    'variant12',
  ];
  const backgroundTypes = ['solid', 'gradientLinear'];

  const generateAvatarUrl = () => {
    return `https://api.dicebear.com/9.x/big-ears/svg?seed=${seed}&backgroundColor=${backgroundColor}&nose=${nose}&rotate=${rotate}&scale=${scale}&radius=${radius}&backgroundType=${backgroundType}`;
  };

  const handleAvatarSelect = () => {
    const newAvatarUrl = generateAvatarUrl();
    setSelectedAvatar(newAvatarUrl);
    onSelectAvatar(newAvatarUrl);
    onClose();
  };

  const handleSeedSelect = (newSeed: string) => {
    setSeed(newSeed);
    setSelectedAvatar(generateAvatarUrl());
  };

  if (!isOpen) return null;

  return (
    <div className='fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50'>
      <div className='bg-background rounded-lg p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto'>
        <div className='flex justify-between items-center mb-4'>
          <h2 className='text-lg font-semibold'>Customize Your Avatar</h2>
          <Button variant='ghost' size='icon' onClick={onClose}>
            <X className='h-4 w-4' />
          </Button>
        </div>

        {/* Preview */}
        <div className='mb-6'>
          <Label className='text-sm font-semibold'>Preview</Label>
          <div className='flex justify-center my-4'>
            <img
              src={generateAvatarUrl()}
              alt='Avatar Preview'
              className='w-32 h-32 rounded-full border-2 border-primary shadow-md'
            />
          </div>
        </div>

        {/* Character Selection (Grid of Avatars) */}
        <div className='mb-6'>
          <Label className='text-sm font-semibold'>Characters</Label>
          <div className='grid grid-cols-5 gap-4 mt-2'>
            {seeds.map((s) => (
              <button
                key={s}
                onClick={() => handleSeedSelect(s)}
                className={`w-16 h-16 rounded-full border-2 transition-all duration-200 ${
                  seed === s
                    ? 'border-primary scale-110'
                    : 'border-transparent hover:border-muted'
                }`}
                title={s}
              >
                <img
                  src={`https://api.dicebear.com/9.x/big-ears/svg?seed=${s}&backgroundColor=${backgroundColor}`}
                  alt={s}
                  className='w-full h-full rounded-full object-cover'
                />
              </button>
            ))}
          </div>
        </div>

        {/* Customization Options */}
        <div className='grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6'>
          {/* Background Color */}
          <div>
            <Label className='text-sm font-semibold'>Background Color</Label>
            <div className='flex flex-wrap gap-2 mt-2'>
              {backgroundColors.map((color) => (
                <button
                  key={color}
                  className={`w-8 h-8 rounded-full border-2 transition-all duration-200 ${
                    backgroundColor === color
                      ? 'border-primary scale-110'
                      : 'border-transparent hover:border-muted'
                  }`}
                  style={{ backgroundColor: `#${color}` }}
                  onClick={() => setBackgroundColor(color)}
                />
              ))}
            </div>
          </div>

          {/* Nose Options */}
          <div>
            <Label className='text-sm font-semibold'>Nose Style</Label>
            <Select value={nose} onValueChange={setNose}>
              <SelectTrigger className='w-full mt-2'>
                <SelectValue placeholder='Select nose style' />
              </SelectTrigger>
              <SelectContent>
                {noseOptions.map((n) => (
                  <SelectItem key={n} value={n}>
                    {n}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Background Type */}
          <div>
            <Label className='text-sm font-semibold'>Background Type</Label>
            <Select value={backgroundType} onValueChange={setBackgroundType}>
              <SelectTrigger className='w-full mt-2'>
                <SelectValue placeholder='Select background type' />
              </SelectTrigger>
              <SelectContent>
                {backgroundTypes.map((type) => (
                  <SelectItem key={type} value={type}>
                    {type === 'solid' ? 'Solid' : 'Gradient Linear'}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Rotation Slider */}
          <div>
            <Label className='text-sm font-semibold'>
              Rotation ({rotate}°)
            </Label>
            <Slider
              value={[rotate]}
              onValueChange={(value) => setRotate(value[0])}
              min={0}
              max={360}
              step={1}
              className='mt-2'
            />
          </div>

          {/* Scale Slider */}
          <div>
            <Label className='text-sm font-semibold'>Scale ({scale}%)</Label>
            <Slider
              value={[scale]}
              onValueChange={(value) => setScale(value[0])}
              min={50}
              max={200}
              step={1}
              className='mt-2'
            />
          </div>

          {/* Radius Slider */}
          <div>
            <Label className='text-sm font-semibold'>Radius ({radius}%)</Label>
            <Slider-English
              value={[radius]}
              onValueChange={(value) => setRadius(value[0])}
              min={0}
              max={50}
              step={1}
              className='mt-2'
            />
          </div>
        </div>

        {/* Action Buttons */}
        <div className='flex gap-2'>
          <Button className='flex-1' onClick={handleAvatarSelect}>
            Save
          </Button>
          <Button variant='secondary' className='flex-1' onClick={onClose}>
            Cancel
          </Button>
        </div>
      </div>
    </div>
  );
};

export default AvatarModal;
