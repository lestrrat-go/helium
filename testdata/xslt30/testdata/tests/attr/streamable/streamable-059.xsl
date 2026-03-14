<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:g="http://www.w3.org/xsl-tests/grouped-transactions"
    xmlns="http://www.w3.org/xsl-tests/grouped-transactions"
    exclude-result-prefixes="g xs"
    version="3.0">
       
  
    <!-- within a streaming template, use distinct-values() on the streamed input -->
     
    <xsl:import-schema namespace="http://www.w3.org/xsl-tests/grouped-transactions" schema-location="grouped-transactions.xsd"/>
  
    <xsl:mode name="s" streamable="yes"/>
         
    <xsl:output method="xml" indent="yes" encoding="UTF-8" />
    
    <xsl:param name="sep" select="'|'" as="xs:string"/>
     
    <xsl:template name="main" match="/">
      <out>
        <xsl:source-document streamable="true" href="grouped-transactions.xml" validation="strict">
          <xsl:apply-templates select="." mode="s"/>
        </xsl:source-document>
      </out>
    </xsl:template>
    
    <xsl:template match="g:account" mode="s">
      <values><xsl:value-of select="distinct-values(.//@value)" separator="{$sep}"/></values>
    </xsl:template>
    
     
    <xsl:template match="text()" mode="s"/>
  
    
</xsl:transform>

