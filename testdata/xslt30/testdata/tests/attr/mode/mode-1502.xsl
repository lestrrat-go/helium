<?xml version="1.0"?> 

<!-- conflicting attributes on xsl:mode -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:mode name="c" on-no-match="shallow-copy" on-multiple-match="fail"/> 
  <xsl:mode name="c" on-no-match="text-only-copy"/>
    
  <xsl:template name="main">
    <out>
      <xsl:apply-templates mode="c"/>
    </out>
  </xsl:template>
  
    
</xsl:stylesheet>