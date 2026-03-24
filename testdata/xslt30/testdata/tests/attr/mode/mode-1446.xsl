<?xml version="1.0"?> 

<!-- Test xsl:mode typed="0" --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
 
  <xsl:mode name="s" on-no-match="shallow-copy" typed="0"/>
    
</xsl:stylesheet>