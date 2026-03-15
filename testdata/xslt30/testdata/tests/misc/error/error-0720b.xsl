<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                        xsl:attribute-set element directly or indirectly references
                     itself via the names contained in the use-attribute-sets
                     attribute.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: circular reference to xsl:attribute-set -->
<?spec xslt#errors?><?error XTSE0720?>


  <xsl:attribute-set name="aset1" use-attribute-sets="aset2">
      <xsl:attribute name="one">1</xsl:attribute>
      <xsl:attribute name="two">2</xsl:attribute>
  </xsl:attribute-set>

  <xsl:attribute-set name="aset2" use-attribute-sets="aset1">
      <xsl:attribute name="three">3</xsl:attribute>
      <xsl:attribute name="four">4</xsl:attribute>
  </xsl:attribute-set>

  <xsl:template match="doc">
      <out>
         <xsl:copy use-attribute-sets="aset1"/>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>

</xsl:stylesheet>
