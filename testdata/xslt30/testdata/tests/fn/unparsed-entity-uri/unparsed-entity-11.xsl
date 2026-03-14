<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" version="2.0">

  <!-- PURPOSE: test that the result of unparsed-entity-uri() is an xs:anyURI -->
  <?spec xslt#unparsed-entity-uri?>
  <xsl:param name="x" select="unparsed-entity-uri('hatch-pic')"/>
  <xsl:template match="doc">   
    <out>
      <xsl:value-of select="$x instance of xs:anyURI"/>     
    </out>
  </xsl:template>
 
</xsl:stylesheet>
