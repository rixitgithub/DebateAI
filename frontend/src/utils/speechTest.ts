export class SpeechRecognitionTest {
  private recognition: SpeechRecognition | null = null;
  private supported: boolean;

  constructor() {
    this.supported =
      'SpeechRecognition' in window || 'webkitSpeechRecognition' in window;

    if (this.supported) {
      const SpeechRecognitionCtor =
        window.SpeechRecognition || window.webkitSpeechRecognition;

      if (SpeechRecognitionCtor) {
        this.recognition = new SpeechRecognitionCtor();
        this.setupRecognition();
      } else {
        console.error('SpeechRecognition constructor not found');
      }
    }
  }

  private setupRecognition() {
    if (!this.recognition) return;

    this.recognition.continuous = true;
    this.recognition.interimResults = true;
    this.recognition.lang = 'en-US';
    this.recognition.maxAlternatives = 1;

    this.recognition.onstart = () => {
      console.log('Speech recognition test started');
    };

    this.recognition.onresult = (event: SpeechRecognitionEvent) => {
      console.log('Speech recognition test result:', event);
      let interimTranscript = '';
      let finalTranscript = '';

      for (let i = event.resultIndex; i < event.results.length; i++) {
        const result = event.results[i];
        const transcript = result[0].transcript;

        if (result.isFinal) {
          finalTranscript += transcript + ' ';
        } else {
          interimTranscript += transcript;
        }
      }

      if (finalTranscript) {
        console.log('Final transcript:', finalTranscript.trim());
      }
      if (interimTranscript) {
        console.log('Interim transcript:', interimTranscript);
      }
    };

    this.recognition.onend = () => {
      console.log('Speech recognition test ended');
    };

    // ✅ Fix typing issue with type assertion
    this.recognition.onerror = (event: Event) => {
      const err = event as unknown as SpeechRecognitionErrorEvent;
      console.error('Speech recognition test error:', err.error, err.message);
    };
  }

  public start(): boolean {
    if (!this.supported || !this.recognition) {
      console.error('Speech recognition not supported');
      return false;
    }

    try {
      this.recognition.start();
      return true;
    } catch (error) {
      console.error('Error starting speech recognition test:', error);
      return false;
    }
  }

  public stop() {
    if (this.recognition) {
      try {
        this.recognition.stop();
      } catch (error) {
        console.error('Error stopping speech recognition test:', error);
      }
    }
  }

  public isSpeechRecognitionSupported(): boolean {
    return this.supported;
  }

  public async checkMicrophonePermission(): Promise<boolean> {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      stream.getTracks().forEach((track) => track.stop());
      return true;
    } catch (error) {
      console.error('Microphone permission check failed:', error);
      return false;
    }
  }
}

// Export a simple test function
export const testSpeechRecognition = () => {
  const test = new SpeechRecognitionTest();

  if (!test.isSpeechRecognitionSupported()) {
    console.error('Speech recognition is not supported in this browser');
    return;
  }

  console.log('Starting speech recognition test...');
  test.start();

  // Stop after 10 seconds
  setTimeout(() => {
    test.stop();
    console.log('Speech recognition test completed');
  }, 10000);
};
