<?xml version="1.0"?> 

<!-- Test initial match selection different from global context item -->


<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:mode name="m"/> 

  <xsl:variable name="bible" select="."/>

  
  <xsl:template match="." mode="m">
    <xsl:result-document href="verse{.}.xml">
      <chapter nr="1" verse="{.}">
        <xsl:copy-of select="$bible/book/chapter[1]/v[current()]"/>
      </chapter>
    </xsl:result-document>
  </xsl:template>
  
    
</xsl:stylesheet>