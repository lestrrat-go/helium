<?xml version="1.0"?> 

<!-- Check that an xsl:mode with no @streamable attribute isn't treated as streamable="no" --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  


  <xsl:mode name="X" streamable="yes"/>  
  <xsl:mode name="X" visibility="public"/>
  
  <xsl:template match="foo[bar='baz']" mode="X">
    <xsl:copy-of select="thing"/>
    <xsl:copy-of select="thung"/>
  </xsl:template>

  
    
</xsl:stylesheet>