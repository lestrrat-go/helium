<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- streaming with xsl:result-document. Also uses position() and selective strip-space -->
   
  <!--<xsl:import-schema schema-location="loans.xsd"/>-->

  <xsl:mode name="s" streamable="yes"/>
  <xsl:mode name="t" streamable="yes"/>
       
  <xsl:output method="xml" indent="no" encoding="UTF-8" />
  <xsl:strip-space elements="Pool"/>
  
    
  <xsl:template name="main" match="/">
    <out>
      <xsl:source-document streamable="true" href="loans.xml" validation="strict">
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
  
  <xsl:template match="Loan" mode="s">
    <xsl:variable name="p" select="position()"/>
    <Loan href="streamable-021-loan{$p}.xml"/>
    <xsl:result-document href="streamable-021-loan{$p}.xml">
      <Loan>
        <xsl:apply-templates mode="s"/>
      </Loan>
    </xsl:result-document>
  </xsl:template>
    
</xsl:transform>

