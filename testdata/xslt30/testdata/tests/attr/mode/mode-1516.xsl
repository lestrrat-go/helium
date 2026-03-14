<?xml version="1.0"?> 

<!-- on-multiple-match="fail" should not apply when matching two branches of a union pattern (proposal in spec bug 30402) -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:mode name="c" on-multiple-match="fail"/> 
    
  <xsl:template name="main">
    <out>
      <xsl:apply-templates mode="c"/>
    </out>
  </xsl:template>
  
  <xsl:template match="para[foo] | para[text()]" mode="c">
    <unambiguous-match/>
  </xsl:template>
  
    
</xsl:stylesheet>