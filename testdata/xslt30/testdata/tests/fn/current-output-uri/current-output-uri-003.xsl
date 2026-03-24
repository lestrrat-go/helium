<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- current-output-uri() within xsl:result-document -->
  
  <xsl:template name="main">
    <out>first</out>
    <xsl:result-document href="second/current-output-uri-003.xml">
    	<out><xsl:value-of select="current-output-uri()"/></out>
    </xsl:result-document>	
  </xsl:template>

</xsl:stylesheet>
