<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE test dynamic call to unparsed-entity-public-id() function, with context in closure -->
<?spec xslt#unparsed-entity-uri?>
  <xsl:template match="doc">
    <xsl:variable name="uepi" select="unparsed-entity-public-id#1"/>
    <out>
      <xsl:for-each select="'watch-pic'">
        <in><xsl:value-of select="$uepi(.) = '-//Textuality//TEXT standard boilerplate//EN'"/></in>
      </xsl:for-each>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
