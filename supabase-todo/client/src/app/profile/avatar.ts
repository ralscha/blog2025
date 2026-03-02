import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  effect,
  inject,
  input,
  output,
  signal,
} from '@angular/core';
import { DomSanitizer, SafeUrl } from '@angular/platform-browser';
import { SupabaseService } from '../supabase.service';

@Component({
  selector: 'app-avatar',
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './avatar.html',
})
export class AvatarComponent {
  private readonly supabase = inject(SupabaseService);
  private readonly sanitizer = inject(DomSanitizer);
  private readonly destroyRef = inject(DestroyRef);

  avatarUrl = input<string | null>(null);
  upload = output<string>();

  avatarSrc = signal<SafeUrl | null>(null);
  uploading = signal(false);
  uploadError = signal<string | null>(null);

  private currentObjectUrl: string | null = null;

  constructor() {
    effect(() => {
      const url = this.avatarUrl();
      if (url) {
        this.downloadImage(url);
      } else {
        this.revokeCurrentUrl();
        this.avatarSrc.set(null);
      }
    });

    this.destroyRef.onDestroy(() => this.revokeCurrentUrl());
  }

  private revokeCurrentUrl() {
    if (this.currentObjectUrl) {
      URL.revokeObjectURL(this.currentObjectUrl);
      this.currentObjectUrl = null;
    }
  }

  private setObjectUrl(blob: Blob) {
    this.revokeCurrentUrl();
    const url = URL.createObjectURL(blob);
    this.currentObjectUrl = url;
    this.avatarSrc.set(this.sanitizer.bypassSecurityTrustUrl(url));
  }

  private async downloadImage(path: string) {
    const { data, error } = await this.supabase.downloadImage(path);
    if (error) {
      console.error('Error downloading avatar:', error.message);
      return;
    }
    if (data instanceof Blob) {
      this.setObjectUrl(data);
    }
  }

  async uploadAvatar(event: Event) {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    this.uploading.set(true);
    this.uploadError.set(null);
    const file = input.files[0];
    const fileExt = file.name.split('.').pop();

    const user = await this.supabase.getUser();
    if (!user) {
      this.uploadError.set('Not authenticated');
      this.uploading.set(false);
      return;
    }
    const filePath = `${user.id}/${crypto.randomUUID()}.${fileExt}`;

    const { error } = await this.supabase.uploadAvatar(filePath, file);

    if (error) {
      this.uploadError.set(error.message);
    } else {
      this.setObjectUrl(file);
      this.upload.emit(filePath);
    }

    this.uploading.set(false);
    input.value = '';
  }
}
