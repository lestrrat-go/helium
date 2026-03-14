<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE test dynamic call to unparsed-entity-uri() function, with context in closure -->
<?spec xslt#unparsed-entity-uri?>
  <xsl:template match="doc">
    <xsl:variable name="ueu" select="unparsed-entity-uri#1"/>
    <out>
      <xsl:for-each select="'hatch-pic'">
        <in><xsl:value-of select="ends-with($ueu(.), '/fn/grafix/OpenHatch.gif')"/></in>
      </xsl:for-each>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
