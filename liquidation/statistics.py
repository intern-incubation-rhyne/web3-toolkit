#!/usr/bin/env python3
"""
Liquidation Data Visualization Script
Creates monthly revenue charts from liquidation data
"""

import json
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
from datetime import datetime, timezone
import seaborn as sns
from pathlib import Path
import numpy as np

def load_data(file_path):
    """Load liquidation data from JSON file"""
    try:
        with open(file_path, 'r') as f:
            data = json.load(f)
        print(f"Loaded {len(data)} records from {file_path}")
        return data
    except FileNotFoundError:
        print(f"File not found: {file_path}")
        return []
    except json.JSONDecodeError:
        print(f"Invalid JSON in file: {file_path}")
        return []

def convert_timestamp_to_date(timestamp):
    """Convert Unix timestamp to datetime"""
    return datetime.fromtimestamp(timestamp, tz=timezone.utc)

def process_data(data):
    """Process data and create monthly aggregates"""
    if not data:
        return pd.DataFrame()
    
    # Convert to DataFrame
    df = pd.DataFrame(data)
    
    # Convert blockTimestamp to datetime
    df['date'] = df['blockTimestamp'].apply(convert_timestamp_to_date)
    
    # Extract month and year
    df['month_year'] = df['date'].dt.to_period('M')
    
    # Convert revenue to ETH (assuming revenue is in wei)
    df['revenue_eth'] = df['revenue'] / 1e18
    
    # Group by month and sum revenue
    monthly_revenue = df.groupby('month_year')['revenue_eth'].sum().reset_index()
    monthly_revenue['month_year'] = monthly_revenue['month_year'].astype(str)
    
    # Convert month_year back to datetime for plotting
    monthly_revenue['date'] = monthly_revenue['month_year'].dt.to_timestamp()
    
    return monthly_revenue, df

def process_morpho_data(data):
    """Process Morpho data with marketId grouping and ranking"""
    if not data:
        return pd.DataFrame(), pd.DataFrame()
    
    # Convert to DataFrame
    df = pd.DataFrame(data)
    
    # Convert blockTimestamp to datetime
    df['date'] = df['blockTimestamp'].apply(convert_timestamp_to_date)
    
    # Extract month and year
    df['month_year'] = df['date'].dt.to_period('M')
    
    # Convert revenue to ETH (assuming revenue is in wei)
    df['revenue_eth'] = df['revenue'] / 1e18
    
    # 1. Sum up total revenue of each marketId
    market_revenue = df.groupby('marketId')['revenue_eth'].sum().reset_index()
    
    # 2. Rank marketIds in revenue descending order
    market_revenue = market_revenue.sort_values('revenue_eth', ascending=False).reset_index(drop=True)
    
    print(f"\n=== MARKET REVENUE RANKING ===")
    for i, row in market_revenue.iterrows():
        print(f"{i+1}. Market {row['marketId'][:10]}... | Revenue: {row['revenue_eth']:.6f} ETH")
    
    # 3. Get top 5 marketIds
    top_5_markets = market_revenue.head(5)['marketId'].tolist()
    
    # 4. Create monthly data with top 5 markets + others
    monthly_market_data = df.groupby(['month_year', 'marketId'])['revenue_eth'].sum().reset_index()
    
    # 5. Pivot data to have markets as columns
    monthly_pivot = monthly_market_data.pivot(index='month_year', columns='marketId', values='revenue_eth').fillna(0)
    
    # 6. Group non-top-5 markets into "Others"
    other_markets = [col for col in monthly_pivot.columns if col not in top_5_markets]
    if other_markets:
        monthly_pivot['Others'] = monthly_pivot[other_markets].sum(axis=1)
        monthly_pivot = monthly_pivot.drop(columns=other_markets)
    
    # 7. Ensure we have all top 5 markets (fill missing with 0)
    for market in top_5_markets:
        if market not in monthly_pivot.columns:
            monthly_pivot[market] = 0
    
    # 8. Reorder columns to have Others last
    if 'Others' in monthly_pivot.columns:
        columns_order = top_5_markets + ['Others']
    else:
        columns_order = top_5_markets
    
    monthly_pivot = monthly_pivot[columns_order]
    
    # 9. Convert month_year back to datetime for plotting
    monthly_pivot = monthly_pivot.reset_index()
    monthly_pivot['date'] = monthly_pivot['month_year'].dt.to_timestamp()
    
    return monthly_pivot, df, top_5_markets

def create_monthly_chart(monthly_data, output_dir="charts"):
    """Create monthly revenue column chart"""
    if monthly_data.empty:
        print("No data to plot")
        return
    
    # Set style - use default matplotlib style for compatibility
    plt.style.use('default')
    sns.set_theme(style="whitegrid")
    
    # Create figure
    fig, ax = plt.subplots(figsize=(16, 10))
    
    # Create bar chart with wider bars
    bars = ax.bar(monthly_data['date'], monthly_data['revenue_eth'], 
                  width=20, color='skyblue', alpha=0.7, edgecolor='navy', linewidth=0.5)
    
    # Customize chart
    ax.set_title('Monthly Liquidation Revenue (Jan 2025 - Present)', 
                fontsize=16, fontweight='bold', pad=20)
    ax.set_xlabel('Month', fontsize=12, fontweight='bold')
    ax.set_ylabel('Revenue (ETH)', fontsize=12, fontweight='bold')
    
    # Format x-axis
    ax.xaxis.set_major_locator(mdates.MonthLocator(interval=1))
    ax.xaxis.set_major_formatter(mdates.DateFormatter('%b %Y'))
    plt.xticks(rotation=45, ha='right')
    
    # Add value labels on bars
    for bar in bars:
        height = bar.get_height()
        if height > 0:
            ax.text(bar.get_x() + bar.get_width()/2., height + height*0.01,
                   f'{height:.4f}', ha='center', va='bottom', fontsize=9)
    
    # Add grid
    ax.grid(True, alpha=0.3, axis='y')
    
    # Adjust layout
    plt.tight_layout()
    
    # Create output directory
    Path(output_dir).mkdir(exist_ok=True)
    
    # Save chart
    output_path = Path(output_dir) / "monthly_revenue_chart.png"
    plt.savefig(output_path, dpi=300, bbox_inches='tight')
    print(f"Chart saved to: {output_path}")
    
    # Show chart
    plt.show()

def create_morpho_monthly_chart(monthly_data, top_5_markets, output_dir="charts"):
    """Create monthly revenue stacked column chart for Morpho data with top 5 markets + others"""
    if monthly_data.empty:
        print("No data to plot")
        return
    
    # Set style - use default matplotlib style for compatibility
    plt.style.use('default')
    sns.set_theme(style="whitegrid")
    
    # Create figure
    fig, ax = plt.subplots(figsize=(18, 12))
    
    # Define colors for each market
    colors = ['#1f77b4', '#ff7f0e', '#2ca02c', '#d62728', '#9467bd', '#8c564b']  # Blue, Orange, Green, Red, Purple, Brown
    
    # Prepare data for stacked bar chart
    dates = monthly_data['date']
    market_columns = [col for col in monthly_data.columns if col not in ['month_year', 'date']]
    
    # Create stacked bars
    bottom = np.zeros(len(dates))
    bars = []
    
    for i, market in enumerate(market_columns):
        values = monthly_data[market]
        bar = ax.bar(dates, values, bottom=bottom, 
                    width=20, color=colors[i % len(colors)], 
                    alpha=0.8, edgecolor='black', linewidth=0.5,
                    label=market[:10] + '...' if len(market) > 10 else market)
        bars.append(bar)
        bottom += values
    
    # Customize chart
    ax.set_title('Monthly Morpho Liquidation Revenue by Market (Top 5 + Others)', 
                fontsize=18, fontweight='bold', pad=20)
    ax.set_xlabel('Month', fontsize=14, fontweight='bold')
    ax.set_ylabel('Revenue (ETH)', fontsize=14, fontweight='bold')
    
    # Format x-axis
    ax.xaxis.set_major_locator(mdates.MonthLocator(interval=1))
    ax.xaxis.set_major_formatter(mdates.DateFormatter('%b %Y'))
    plt.xticks(rotation=45, ha='right')
    
    # Add legend
    ax.legend(title='Markets', bbox_to_anchor=(1.05, 1), loc='upper left')
    
    # Add grid
    ax.grid(True, alpha=0.3, axis='y')
    
    # Adjust layout to accommodate legend
    plt.tight_layout()
    
    # Create output directory
    Path(output_dir).mkdir(exist_ok=True)
    
    # Save chart
    output_path = Path(output_dir) / "morpho_monthly_revenue_by_market.png"
    plt.savefig(output_path, dpi=300, bbox_inches='tight')
    print(f"Chart saved to: {output_path}")
    
    # Show chart
    plt.show()

def create_detailed_analysis(df):
    """Create detailed analysis of the data"""
    if df.empty:
        return
    
    print("\n=== DETAILED ANALYSIS ===")
    print(f"Total records: {len(df)}")
    print(f"Date range: {df['date'].min().strftime('%Y-%m-%d')} to {df['date'].max().strftime('%Y-%m-%d')}")
    print(f"Total revenue: {df['revenue_eth'].sum():.6f} ETH")
    print(f"Average revenue per liquidation: {df['revenue_eth'].mean():.6f} ETH")
    print(f"Median revenue per liquidation: {df['revenue_eth'].median():.6f} ETH")
    print(f"Max revenue: {df['revenue_eth'].max():.6f} ETH")
    print(f"Min revenue: {df['revenue_eth'].min():.6f} ETH")
    
    # Top 10 liquidations by revenue
    print("\n=== TOP 10 LIQUIDATIONS BY REVENUE ===")
    top_10 = df.nlargest(10, 'revenue_eth')[['date', 'transactionHash', 'revenue_eth']]
    for _, row in top_10.iterrows():
        print(f"{row['date'].strftime('%Y-%m-%d %H:%M')} | {row['transactionHash'][:10]}... | {row['revenue_eth']:.6f} ETH")

def create_additional_charts(df, output_dir="charts"):
    """Create additional visualization charts"""
    if df.empty:
        return
    
    # Create output directory
    Path(output_dir).mkdir(exist_ok=True)
    
    # 1. Daily revenue trend
    daily_revenue = df.groupby(df['date'].dt.date)['revenue_eth'].sum().reset_index()
    daily_revenue['date'] = pd.to_datetime(daily_revenue['date'])
    
    plt.figure(figsize=(14, 6))
    plt.plot(daily_revenue['date'], daily_revenue['revenue_eth'], marker='o', linewidth=2, markersize=4)
    plt.title('Daily Liquidation Revenue Trend', fontsize=14, fontweight='bold')
    plt.xlabel('Date', fontsize=12)
    plt.ylabel('Revenue (ETH)', fontsize=12)
    plt.grid(True, alpha=0.3)
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig(Path(output_dir) / "daily_revenue_trend.png", dpi=300, bbox_inches='tight')
    plt.show()
    
    # 2. Revenue distribution histogram
    plt.figure(figsize=(12, 6))
    plt.hist(df['revenue_eth'], bins=50, alpha=0.7, color='lightcoral', edgecolor='black')
    plt.title('Revenue Distribution', fontsize=14, fontweight='bold')
    plt.xlabel('Revenue (ETH)', fontsize=12)
    plt.ylabel('Frequency', fontsize=12)
    plt.grid(True, alpha=0.3)
    plt.tight_layout()
    plt.savefig(Path(output_dir) / "revenue_distribution.png", dpi=300, bbox_inches='tight')
    plt.show()

def main():
    """Main function"""
    print("=== LIQUIDATION DATA VISUALIZATION ===\n")
    data = load_data("data/mainnet_morpho_logs_with_revenue.json")
    
    if not data:
        print("No data found. Please ensure the data file exists.")
        return
    
    # Check if data has marketId field (Morpho data)
    if 'marketId' in data[0]:
        print("Processing Morpho data with marketId grouping...")
        monthly_data, df, top_5_markets = process_morpho_data(data)
        
        if monthly_data.empty:
            print("No valid data to process")
            return
        
        # Create Morpho-specific visualizations
        print("\nCreating monthly revenue chart by market...")
        create_morpho_monthly_chart(monthly_data, top_5_markets)
        
    else:
        print("Processing standard liquidation data...")
        monthly_data, df = process_data(data)
        
        if monthly_data.empty:
            print("No valid data to process")
            return
        
        # Create standard visualizations
        print("\nCreating monthly revenue chart...")
        create_monthly_chart(monthly_data)
    
    # # Create additional charts
    # print("\nCreating additional charts...")
    # create_additional_charts(df)
    
    # # Print detailed analysis
    # create_detailed_analysis(df)
    
    print("\n=== VISUALIZATION COMPLETE ===")
    print("Charts have been saved to the 'charts' directory")

if __name__ == "__main__":
    main()
