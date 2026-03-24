<?xml version="1.0"?> 

<!-- Override on-no-match --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <xsl:import href="mode-0001.xsl"/>


  <xsl:mode name="s" on-no-match="fail"/>
  
    
</xsl:stylesheet>