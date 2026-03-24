<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    version="3.0">
       
  
  <!-- Two local variables in streaming template -->
   
  <xsl:import-schema schema-location="loans.xsd"/>

  <xsl:mode name="s" streamable="yes"/>

  
  <xsl:param name="use-prefix" select="true()"/>
    
    
  <xsl:template name="main" match="/">
    <myroot>
      <xsl:source-document streamable="true" href="loans.xml">
        <xsl:apply-templates select="." mode="s"/>
      </xsl:source-document>  
    </myroot>
  </xsl:template>
  
  <xsl:template match="Deal" mode="s">
    <ACERequest>
      <Deal>
        <!-- MHK: rearranged this to make it streamable -->
        <xsl:apply-templates mode="s"/>
      </Deal>
    </ACERequest>
  </xsl:template>
  
  <xsl:template match="Deal/DealIdentifier" mode="s" priority="5">
   <DealIdentifier>
      <xsl:value-of select="." />
   </DealIdentifier>
  </xsl:template>
  
  <xsl:template match="Deal/ACERequestType" mode="s" priority="5">
   <ACERequestType>
      <xsl:value-of select="." />
   </ACERequestType>
  </xsl:template>
  
  <xsl:template match="Deal/*" mode="s" priority="4"/>
      
  <xsl:template match="Pool" mode="s" priority="6">
    <Pool>
       <xsl:apply-templates  mode="s"/>
    </Pool>
  </xsl:template>
  
  <xsl:template match="Pool/PoolIdentifier" mode="s" priority="5">
    <!-- The original code output this AFTER the Loan information -->
    <PoolIdentifier>
        <xsl:value-of select="." />
    </PoolIdentifier>
  </xsl:template>
  
  <xsl:template match="Pool/*" mode="s" priority="4"/>
      
   <xsl:template match="Loan" mode="s" priority="6">
     <xsl:variable name="prefix" select="if ($use-prefix) then '#' else ''"/>
     <xsl:variable name="Loan">
       <xsl:copy-of select="."/>
     </xsl:variable>
      <Loan>
      <xsl:apply-templates select="$Loan/Loan/LoanStatus" />
      <xsl:apply-templates select="$Loan/Loan/LoanToValueRatio" />
      <xsl:apply-templates select="$Loan/Loan/Property" />
      <FinancialInstrumentIdentifier>
        <xsl:value-of select="concat($prefix, $Loan/Loan/FinancialInstrumentIdentifier)" />
      </FinancialInstrumentIdentifier>
      <LoanPurposeType>
        <xsl:value-of select="concat($prefix, $Loan/Loan/LoanPurposeType)" />
      </LoanPurposeType>
      <NoteDate>
        <xsl:value-of select="concat($prefix, $Loan/Loan/NoteDate)" />
      </NoteDate>
    </Loan>
   </xsl:template>
   
  <xsl:template match="Property">
    <Property>
      <xsl:apply-templates select="Address" />
      <xsl:apply-templates select="PropertyCharacteristic" />
      <xsl:apply-templates select="PropertyPurchase" />
      <xsl:apply-templates select="PropertyValuation" />
    </Property>
  </xsl:template>
  <xsl:template match="PropertyValuation">
    <PropertyValuation>
      <xsl:attribute name="PropertyAppraisedValueAmount">
        <xsl:value-of select="PropertyAppraisedValueAmount" />
      </xsl:attribute>
    </PropertyValuation>
  </xsl:template>
  <xsl:template match="PropertyPurchase">
    <PropertyPurchase>
      <xsl:attribute name="PurchasePriceAmount">
        <xsl:value-of select="PurchasePriceAmount" />
      </xsl:attribute>
    </PropertyPurchase>
  </xsl:template>
  <xsl:template match="PropertyCharacteristic">
    <PropertyCharacteristic>
      <xsl:attribute name="PropertyCharacteristicType">
        <xsl:value-of select="PropertyCharacteristicType" />
      </xsl:attribute>
      <xsl:attribute name="PropertyCharacteristicValue">
        <xsl:value-of select="PropertyCharacteristicValue" />
      </xsl:attribute>
    </PropertyCharacteristic>
  </xsl:template>
  <xsl:template match="Address">
    <Address>
      <xsl:attribute name="PostalCode">
        <xsl:value-of select="PostalCode" />
      </xsl:attribute>
      <xsl:attribute name="State">
        <xsl:value-of select="State" />
      </xsl:attribute>
    </Address>
  </xsl:template>
  <xsl:template match="LoanToValueRatio">
    <LoanToValueRatio>
      <xsl:apply-templates select="LoanToValueRule" />
      <LoanToValueRatioPercent>
        <xsl:value-of select="LoanToValueRatioPercent" />
      </LoanToValueRatioPercent>
    </LoanToValueRatio>
  </xsl:template>
  <xsl:template match="LoanToValueRule">
    <LoanToValueRule>
      <xsl:attribute name="LoanScopeType">
        <xsl:value-of select="LoanScopeType" />
      </xsl:attribute>
      <xsl:attribute name="TimeType">
        <xsl:value-of select="TimeType" />
      </xsl:attribute>
      <xsl:attribute name="PercentIndicator">
        <xsl:value-of select="PercentIndicator" />
      </xsl:attribute>
    </LoanToValueRule>
  </xsl:template>
  <xsl:template match="LoanStatus">
    <LoanStatus>
      <xsl:attribute name="LoanStatusType">
        <xsl:value-of select="LoanStatusType" />
      </xsl:attribute>
      <xsl:attribute name="UnpaidPrincipalBalanceAmount">
        <xsl:value-of select="UnpaidPrincipalBalanceAmount" />
      </xsl:attribute>
    </LoanStatus>
  </xsl:template>
</xsl:transform>

