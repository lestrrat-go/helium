<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
                
  <!-- Variable is a childless document node -->
  
  
  
  <xsl:variable name="data">
    <xsl:sequence select="()"/>
  </xsl:variable>
  
 
  <xsl:template name="main">
    <out ok="{empty($data/child::node())}"/>   
  </xsl:template>

  
</xsl:stylesheet>
