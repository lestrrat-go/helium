<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the value of
                     the use-attribute-sets attribute of an xsl:copy,
                        xsl:element, or xsl:attribute-set element,
                     or the xsl:use-attribute-sets attribute of a literal result element, is not a
                     whitespace-separated sequence of EQNames, or
                     if it contains a QName that does not match the name attribute of
                     any xsl:attribute-set declaration in the stylesheet.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: reference to unknown xsl:attribute-set -->
<?spec xslt#errors?><?error XTSE0710?>

  <xsl:attribute-set name="aset1" use-attribute-sets="aset2">
      <xsl:attribute name="one">1</xsl:attribute>
      <xsl:attribute name="two">2</xsl:attribute>
  </xsl:attribute-set>

  <xsl:attribute-set name="aset2">
      <xsl:attribute name="three">3</xsl:attribute>
      <xsl:attribute name="four">4</xsl:attribute>
  </xsl:attribute-set>

  <xsl:template match="doc">
      <out>
         <xsl:copy use-attribute-sets="aset1 aset2 aset3"/>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>

</xsl:stylesheet>
