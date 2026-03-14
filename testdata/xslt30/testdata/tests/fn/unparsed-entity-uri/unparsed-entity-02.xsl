<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE test unparsed-entity-uri() function -->
<?spec xslt#unparsed-entity-uri?>
  <xsl:template match="doc">
    <out>
      <xsl:value-of select="ends-with(unparsed-entity-uri('hatch-pic'), '/fn/grafix/OpenHatch.gif')"/>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
