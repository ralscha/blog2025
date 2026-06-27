import { Component, computed, inject, signal } from '@angular/core';
import { email, FormField, FormRoot, form, minLength, required } from '@angular/forms/signals';
import { Router, RouterLink } from '@angular/router';
import { SupabaseService } from '../supabase.service';

@Component({
  selector: 'app-sign-up',
  imports: [FormField, FormRoot, RouterLink],
  templateUrl: './sign-up.html',
})
export class SignUpComponent {
  private readonly supabase = inject(SupabaseService);
  private readonly router = inject(Router);

  loading = signal(false);
  errorMessage = signal<string | null>(null);
  successMessage = signal<string | null>(null);
  submitted = signal(false);

  readonly signUpModel = signal({
    email: '',
    password: '',
    confirmPassword: '',
  });
  readonly form = form(this.signUpModel, (path) => {
    required(path.email);
    email(path.email);
    required(path.password);
    minLength(path.password, 8);
    required(path.confirmPassword);
  });
  readonly passwordMismatch = computed(() => {
    const { password, confirmPassword } = this.signUpModel();
    return confirmPassword.length > 0 && password !== confirmPassword;
  });

  async onSubmit() {
    this.submitted.set(true);
    if (!this.form().valid() || this.passwordMismatch()) return;

    this.loading.set(true);
    this.errorMessage.set(null);
    this.successMessage.set(null);

    const { email, password } = this.signUpModel();
    const { error } = await this.supabase.signUp(email, password);

    this.loading.set(false);

    if (error) {
      this.errorMessage.set(error.message);
      return;
    }

    this.successMessage.set('Account created! Check your email to confirm, then sign in.');
    this.signUpModel.set({ email: '', password: '', confirmPassword: '' });
    this.submitted.set(false);

    setTimeout(() => this.router.navigate(['/sign-in']), 2000);
  }
}
