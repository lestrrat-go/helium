<?xml version="1.0"?> 

<!-- Test initial match selection different from global context item -->


<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:mode name="m"/> 
  
  <xsl:strip-space elements="*"/>
  <xsl:preserve-space elements="v"/>

  
  <xsl:template match="v" mode="m">
    <xsl:result-document href="verse{position()}.xml">
      <chapter nr="1" verse="{position()}" book="{/*/node()[2]}">
        <xsl:copy-of select="current()"/>
      </chapter>
    </xsl:result-document>
  </xsl:template>
  
    
</xsl:stylesheet>