<?xml version="1.0"?> 

<!-- Another Test for white-space -->
 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="Q{}test1"/>
  
  <xsl:template match="doc">
    <out>
	    <xsl:apply-templates select="*"/>
    </out>
  </xsl:template>
   
</xsl:stylesheet>