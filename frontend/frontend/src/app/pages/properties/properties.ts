import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal } from '@angular/core';
import { PropertyCard } from '../../shared/components/property-card/property-card';
import { PropertyService } from '../../shared/services/propertyService';
import { FormBuilder } from '@angular/forms';
import { HotToastService } from '@ngneat/hot-toast';
import { PropertyResponseModel } from '../../shared/models/properties-model';

@Component({
  selector: 'app-properties',
  imports: [CommonModule, PropertyCard],
  templateUrl: './properties.html',
  styleUrl: './properties.css',
})
export class Properties implements OnInit {
  fb = inject(FormBuilder);
  propertyService = inject(PropertyService);
  toastService = inject(HotToastService);

  usr = JSON.parse(localStorage.getItem('uWfUsr') || '');
  userDetails = signal<any>(this.usr);

  isLayout = signal<boolean>(true);
  properties = signal<PropertyResponseModel[]>([]);

  ngOnInit(): void {
    this.getAllProperties();
  }
  toggleLayout() {
    this.isLayout.set(!this.isLayout());
  }

  getAllProperties() {
    const loadingToast = this.toastService.loading('Processing...');

    this.propertyService.getAllProperty().subscribe({
      next: (res) => {
        loadingToast.close();
        this.properties.set(res);
      },
      error: (err) => {
        loadingToast.close();

        this.toastService.error(
          `Something went wrong feting properties! ${err.error.error}!!`,
          {
            duration: 2000,
          }
        );
      },
    });
  }
}
