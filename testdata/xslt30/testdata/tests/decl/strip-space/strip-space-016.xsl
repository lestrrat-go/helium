<?xml version="1.0"?> 

<!-- Another Test for white-space -->
  
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" xmlns:ns1="http://ns1/">
  <xsl:strip-space elements="ns1:*"/>
  
  <xsl:template match="doc">
    <out>
	    <xsl:apply-templates select="*"/>
    </out>
  </xsl:template>
   
</xsl:stylesheet>