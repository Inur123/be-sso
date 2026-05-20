package utils

import (
	"fmt"
	"net/smtp"

	"sso.pelajarnumagetan.or.id/internal/config"
)

func SendEmail(to, subject, body string) error {
	cfg := config.Get()

	// Jika SMTP email belum di-set di env, abaikan pengiriman (skip/log) agar saat dev tidak error jika env kosong
	if cfg.SMTPEmail == "" || cfg.SMTPPassword == "" {
		fmt.Printf("[dev-email] Kirim email ke %s\nSubject: %s\nBody:\n%s\n", to, subject, body)
		return nil
	}

	auth := smtp.PlainAuth("", cfg.SMTPEmail, cfg.SMTPPassword, cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	// Format template email HTML / Plain Text
	header := make(map[string]string)
	header["From"] = fmt.Sprintf("%s <%s>", cfg.AppName, cfg.SMTPEmail)
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	err := smtp.SendMail(addr, auth, cfg.SMTPEmail, []string{to}, []byte(message))
	if err != nil {
		fmt.Printf("Gagal mengirim email ke %s: %v\n", to, err)
		return err
	}

	return nil
}

func SendVerificationEmail(toName, toEmail, token string) error {
	cfg := config.Get()
	// Gunakan AppURL dari config untuk tautan verifikasi di frontend Next.js
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", cfg.AppURL, token)

	subject := "Verifikasi Akun SSO IPNU-IPPNU Magetan"
	body := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e5e7eb; rounded: 12px;">
			<h2 style="color: #10b981; margin-bottom: 16px;">Halo, %s!</h2>
			<p style="color: #374151; line-height: 1.5; margin-bottom: 24px;">
				Terima kasih telah mendaftar di <strong>SSO IPNU-IPPNU Magetan</strong>. Silakan verifikasi email Anda dengan menekan tombol di bawah ini agar dapat masuk ke dalam dashboard.
			</p>
			<div style="text-align: center; margin-bottom: 24px;">
				<a href="%s" style="background-color: #10b981; color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px; font-weight: bold; display: inline-block;">
					Verifikasi Email Saya
				</a>
			</div>
			<p style="color: #6b7280; font-size: 14px; line-height: 1.5; margin-bottom: 16px;">
				Jika tombol di atas tidak bekerja, Anda juga dapat membuka link berikut pada browser Anda:
				<br/>
				<a href="%s" style="color: #10b981;">%s</a>
			</p>
			<hr style="border: 0; border-top: 1px solid #e5e7eb; margin: 24px 0;" />
			<p style="color: #9ca3af; font-size: 12px; text-align: center;">
				Email ini dikirim secara otomatis oleh sistem SSO IPNU-IPPNU Magetan. Tolong jangan membalas email ini.
			</p>
		</div>
	`, toName, verifyURL, verifyURL, verifyURL)

	return SendEmail(toEmail, subject, body)
}

func SendResetPasswordEmail(toName, toEmail, token string) error {
	cfg := config.Get()
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", cfg.AppURL, token)

	subject := "Reset Password Akun SSO IPNU-IPPNU Magetan"
	body := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e5e7eb; border-radius: 12px;">
			<h2 style="color: #10b981; margin-bottom: 16px;">Halo, %s!</h2>
			<p style="color: #374151; line-height: 1.5; margin-bottom: 24px;">
				Anda menerima email ini karena ada permintaan untuk mengatur ulang password akun Anda di <strong>SSO IPNU-IPPNU Magetan</strong>. Silakan klik tombol di bawah ini untuk mereset password Anda:
			</p>
			<div style="text-align: center; margin-bottom: 24px;">
				<a href="%s" style="background-color: #10b981; color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px; font-weight: bold; display: inline-block;">
					Reset Password Saya
				</a>
			</div>
			<p style="color: #ef4444; font-size: 13px; font-weight: bold; margin-bottom: 16px;">
				Tautan ini hanya berlaku selama 15 menit demi keamanan akun Anda.
			</p>
			<p style="color: #6b7280; font-size: 14px; line-height: 1.5; margin-bottom: 16px;">
				Jika tombol di atas tidak bekerja, Anda juga dapat membuka link berikut pada browser Anda:
				<br/>
				<a href="%s" style="color: #10b981;">%s</a>
			</p>
			<p style="color: #9ca3af; font-size: 13px; margin-top: 24px;">
				Jika Anda tidak merasa mengajukan permintaan ini, abaikan email ini dan password Anda tidak akan berubah.
			</p>
			<hr style="border: 0; border-top: 1px solid #e5e7eb; margin: 24px 0;" />
			<p style="color: #9ca3af; font-size: 12px; text-align: center;">
				Email ini dikirim secara otomatis oleh sistem SSO IPNU-IPPNU Magetan. Tolong jangan membalas email ini.
			</p>
		</div>
	`, toName, resetURL, resetURL, resetURL)

	return SendEmail(toEmail, subject, body)
}
