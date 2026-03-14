<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                     XSLT-defined element is used in a context where it is not permitted, if a
                        required attribute is omitted, or if the content of the
                     element does not correspond to the content that is allowed for the element.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:decimal-format (not top level) -->
<?spec xslt#errors?><?error XTDE0010?>

   <xsl:decimal-format name="dec-format" decimal-separator="."/>

  <xsl:template match="doc" name="this">
      <xsl:decimal-format name="wrong-place" decimal-separator=":"/>
      <out>
         <xsl:value-of select="format-number(1234.56, '####0.00', 'dec-format')"/>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>
  

</xsl:stylesheet>
