package kosmixutil

import "runtime"

func GetShell() string {
	switch os := runtime.GOOS; os {
	case "windows":
		return "C:\\Windows\\System32\\cmd.exe"
	case "darwin":
		return "/bin/bash"
	case "linux":
		return "/bin/bash"
	default:
		panic("unsupported OS")
	}
}

// libx264 -> transcode work, convert work
// return []string{"-c:v", "libx264", "-preset", "ultrafast", "-tune", "zerolatency", "-pix_fmt", "yuv420p", "-profile:v", "baseline", "-crf", "22"}
// h264_nvenc -> transcode work, convert work
// return []string{"-c:v", "h264_nvenc", "-preset", "p1", "-pix_fmt", "yuv420p", "-profile:v", "baseline", "-crf", "22"}
func GetEncoderSettings(encoder string) []string {
	switch encoder {
	case "libx264":
		return []string{"-c:v", "libx264", "-preset", "veryfast", "-tune", "zerolatency", "-pix_fmt", "yuv420p", "-profile:v", "baseline"}
	case "h264_nvenc":
		return []string{"-c:v", "h264_nvenc", "-preset", "p1", "-pix_fmt", "yuv420p", "-profile:v", "baseline"}
	case "libvpx-vp9":
		return []string{"-c:v", "libvpx-vp9", "-deadline", "realtime", "-cpu-used", "0", "-row-mt", "1", "-pix_fmt", "yuv420p", "-b:v", "0", "-crf", "30"}
	case "libvpx":
		return []string{"-c:v", "libvpx", "-deadline", "realtime", "-cpu-used", "0", "-row-mt", "1", "-pix_fmt", "yuv420p", "-b:v", "0", "-crf", "30"}
	default:
		panic("unsupported encoder")
	}
}
