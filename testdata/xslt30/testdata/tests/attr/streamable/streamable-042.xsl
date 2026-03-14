<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
    <!-- within a streaming template, test existence of a descendant node -->
     
    <xsl:mode streamable="yes"/>
         
    <xsl:output method="xml" indent="yes" encoding="UTF-8" />
    
    <xsl:strip-space elements="*"/>
      
    <xsl:template name="main">
      <out>
        <xsl:source-document streamable="true" href="loans.xml" validation="strict">
          <xsl:apply-templates select="."/>
        </xsl:source-document>
      </out>
    </xsl:template>
     
    <xsl:template match="*">
      <xsl:copy>
        <xsl:copy-of select="@*"/>
        <xsl:apply-templates/>
      </xsl:copy>
    </xsl:template>
    
    <xsl:template match="Loan">
      <exists><xsl:value-of select="exists(.//PercentIndicator)"/></exists>
    </xsl:template>
      
</xsl:transform>

