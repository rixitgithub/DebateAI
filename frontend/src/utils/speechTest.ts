// Speech Recognition Test Utility
export class SpeechRecognitionTest {
  private recognition: any;
  private isSupported: boolean;

  constructor() {
    this.isSupported = "SpeechRecognition" in window || "webkitSpeechRecognition" in window;
    
    if (this.isSupported) {
      const SpeechRecognition = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
      this.recognition = new SpeechRecognition();
      this.setupRecognition();
    }
  }

  private setupRecognition() {
    this.recognition.continuous = true;
    this.recognition.interimResults = true;
    this.recognition.lang = "en-US";
    this.recognition.maxAlternatives = 1;

    this.recognition.onstart = () => {
      console.log("Speech recognition test started");
    };

    this.recognition.onresult = (event: any) => {
      console.log("Speech recognition test result:", event);
      let interimTranscript = "";
      let finalTranscript = "";

      for (let i = event.resultIndex; i < event.results.length; i++) {
        const result = event.results[i];
        const transcript = result[0].transcript;
        
        if (result.isFinal) {
          finalTranscript += transcript + " ";
        } else {
          interimTranscript += transcript;
        }
      }

      if (finalTranscript) {
        console.log("Final transcript:", finalTranscript.trim());
      }
      if (interimTranscript) {
        console.log("Interim transcript:", interimTranscript);
      }
    };

    this.recognition.onend = () => {
      console.log("Speech recognition test ended");
    };

    this.recognition.onerror = (event: any) => {
      console.error("Speech recognition test error:", event.error);
    };
  }

  public start() {
    if (!this.isSupported) {
      console.error("Speech recognition not supported");
      return false;
    }

    try {
      this.recognition.start();
      return true;
    } catch (error) {
      console.error("Error starting speech recognition test:", error);
      return false;
    }
  }

  public stop() {
    if (this.recognition) {
      try {
        this.recognition.stop();
      } catch (error) {
        console.error("Error stopping speech recognition test:", error);
      }
    }
  }

  public isSupported() {
    return this.isSupported;
  }

  public async checkMicrophonePermission(): Promise<boolean> {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      stream.getTracks().forEach(track => track.stop());
      return true;
    } catch (error) {
      console.error("Microphone permission check failed:", error);
      return false;
    }
  }
}

// Export a simple test function
export const testSpeechRecognition = () => {
  const test = new SpeechRecognitionTest();
  
  if (!test.isSupported()) {
    console.error("Speech recognition is not supported in this browser");
    return;
  }

  console.log("Starting speech recognition test...");
  test.start();

  // Stop after 10 seconds
  setTimeout(() => {
    test.stop();
    console.log("Speech recognition test completed");
  }, 10000);
};
