<?xml version="1.0" encoding="UTF-8"?>
<!--In general, a circularity in a stylesheet is a non-recoverable dynamic error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:variable - circular reference -->
<?spec xslt#errors?><?error XTDE0640?>

   <xsl:variable name="x" select="$y+1"/>

  <xsl:template match="/">
      <out val="{$z}">
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>

   <xsl:variable name="y" select="$z+1"/>
   <xsl:variable name="z" select="$x+1"/>

</xsl:stylesheet>
