<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:g="http://www.w3.org/xsl-tests/grouped-transactions-e"
    xmlns="http://www.w3.org/xsl-tests/grouped-transactions-e"
    exclude-result-prefixes="g xs"
    version="3.0">
       
  
    <!-- within a streaming template, calculate min() of numeric values -->
     
    <xsl:import-schema namespace="http://www.w3.org/xsl-tests/grouped-transactions-e" schema-location="grouped-transactions-e.xsd"/>
  
    <xsl:mode name="s" streamable="yes"/>
         
    <xsl:output method="xml" indent="yes" encoding="UTF-8" />
     
    <xsl:template name="main" match="/">
      <out>
        <xsl:source-document streamable="true" href="grouped-transactions-e.xml" validation="strict">
          <xsl:apply-templates select="." mode="s"/>
        </xsl:source-document>
      </out>
    </xsl:template>
    
    <xsl:template match="*" mode="s">
      <xsl:copy>
        <xsl:copy-of select="@*"/>
        <xsl:apply-templates mode="s"/>
      </xsl:copy>
    </xsl:template>
    
    <xsl:template match="g:value" mode="s">
      <min><xsl:value-of select="format-number(min(.), '0.00')"/></min>
    </xsl:template>
      
</xsl:transform>

