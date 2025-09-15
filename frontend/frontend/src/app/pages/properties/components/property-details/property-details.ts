import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal } from '@angular/core';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { HotToastService } from '@ngneat/hot-toast';
import { PropertyService } from '../../../../shared/services/propertyService';
import { PropertyResponseModel } from '../../../../shared/models/properties-model';

@Component({
  selector: 'app-property-details',
  imports: [CommonModule, RouterModule],
  templateUrl: './property-details.html',
  styleUrl: './property-details.css',
})
export class PropertyDetails implements OnInit {
  private propertyService = inject(PropertyService);
  private toastService = inject(HotToastService);
  private route = inject(ActivatedRoute);

  property = signal<PropertyResponseModel>({});
  selectedImage = signal<string>('');

  ngOnInit(): void {
    this.getProperty();
  }
  getProperty() {
    const loadingToast = this.toastService.loading('Processing...');

    const id = this.route.snapshot.paramMap.get('id');
    if (!id) {
      this.toastService.error('No property ID found in route');
      return;
    }

    this.propertyService.getSingleProperty(id).subscribe({
      next: (res) => {
        loadingToast.close();
        this.property.set(res);
      },
      error: () => {
        loadingToast.close();
        this.toastService.error('Failed to fetch property details');
      },
    });
  }

  selectImage(image: string) {
    this.selectedImage.set(image);
  }
}
