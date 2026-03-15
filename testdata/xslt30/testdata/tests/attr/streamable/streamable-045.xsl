<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:g="http://www.w3.org/xsl-tests/grouped-transactions-e"
    xmlns="http://www.w3.org/xsl-tests/grouped-transactions-e"
    exclude-result-prefixes="g xs"
    version="3.0">
       
  
    <!-- within a streaming template, get typed value of context item into a variable -->
     
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
      <xsl:variable name="value" as="xs:decimal*">
        <xsl:sequence select="data(.)"/>
      </xsl:variable>
      <sum><xsl:value-of select="format-number(sum($value), '0.00')"/></sum>
      <avg><xsl:value-of select="format-number(avg($value), '0.00')"/></avg>
    </xsl:template>
    
</xsl:transform>

