<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<xsl:output encoding="iso-8859-1"/>

  <!-- PURPOSE: xsl:number applied to non-number, with backwards compatibility -->
  <xsl:template match="doc">
    <out>
        <xsl:variable name="empty" select="'fizz'"/>
        <!--<v2><xsl:number value="$empty" format="[1]"/></v2>-->
        <v1 xsl:version="1.0"><xsl:number value="$empty"/></v1>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
